package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"text/template"

	"github.com/bndr/gopencils"
	"github.com/docopt/docopt-go"
	"github.com/op/go-logging"
	"github.com/seletskiy/tplutil"
)

var (
	reStashURL = regexp.MustCompile(
		`https?://(.*)/` +
			`((users|projects)/([^/]+))` +
			`/repos/([^/]+)` +
			`/pull-requests/(\d+)`)
)

var configPath = os.Getenv("HOME") + "/.config/ash/ashrc"

var logger = logging.MustGetLogger("main")

const logFormat = "%{color}%{time:15:04:05.00} [%{level:.4s}] %{message}%{color:reset}"

type CmdLineArgs string

func parseCmdLine(cmd []string) (map[string]interface{}, error) {
	help := `Atlassian Stash Reviewer.

Most convient usage is specify pull request url and file you want to review:
  ash review http://stash.local/projects/.../repos/.../pull-requests/... file

However, you can set up --host and --project flags in ~/.config/ash/ashrc file
and access pull requests by shorthand commands:
  ash review proj/mycoolrepo/1  # if --host is given
  ash review mycoolrepo/1       # if --host and --project is given

Ash then open $EDITOR for commenting on pull request.

You can add comments by just specifying them after line you want to comment,
beginning with '# '.

You can delete comment by deleting it from file, and, of course, modify comment
you own by modifying it in the file.

After you finish your edits, just save file and exit from editor. Ash will
apply all changes made to the review.

If <file-name> is omitted, ash welcomes you to review the overview.

'ls' command can be used to list all files changed in pull request.

Usage:
  ash [options] <pull-request> review [<file-name>]
  ash [options] <pull-request> ls
  ash -h | --help

Options:
  -h --help         Show this help.
  -u --user=<user>  Stash username.
  -p --pass=<pass>  Stash password. You want to set this flag in .ashrc file.
  -e=<editor>       Editor to use. This has priority over $EDITOR env var.
  --debug=<level>   Verbosity [default: 0].
  --url=<url>       Template URL where pull requests are available.
                    Usually you do not need to change that value.
                    [default: /{{.Ns}}/{{.Proj}}/repos/{{.Repo}}/pull-requests/{{.Pr}}]
  --host=<host>     Stash host name. Change to hostname your stash is located.
  --project=<proj>  Use to specify default project that can be used when serching
                    pull requests. Can be set in either <project> or
                    <project>/<repo> format.
`

	args, err := docopt.Parse(help, cmd, true, "0.1 beta", false)

	return args, err
}

func main() {
	logging.SetBackend(logging.NewLogBackend(os.Stderr, "", 0))
	logging.SetFormatter(logging.MustStringFormatter(logFormat))

	rawArgs := mergeArgsWithConfig(configPath)

	args, _ := parseCmdLine(rawArgs)

	logLevels := []logging.Level{
		logging.WARNING,
		logging.INFO,
		logging.DEBUG,
	}

	requestedLogLevel := int64(0)
	if args["--debug"] != nil {
		requestedLogLevel, _ = strconv.ParseInt(args["--debug"].(string), 10, 16)
	}

	for _, lvl := range logLevels[:requestedLogLevel+1] {
		logging.SetLevel(lvl, "main")
	}

	logger.Debug("cmd line args: %s", CmdLineArgs(fmt.Sprintf("%s", rawArgs)))

	url := ""
	prId := args["<pull-request>"].(string)
	if strings.HasPrefix(prId, "http:") || strings.HasPrefix(prId, "https:") {
		url = prId
	} else {
		url = makeUrl(args)
	}

	logger.Debug("url to accessing Stash: %s", url)

	if args["--user"] == nil || args["--pass"] == nil {
		fmt.Println(
			"--user and --pass should be specified.")
		os.Exit(1)
	}

	matches := reStashURL.FindStringSubmatch(url)
	if len(matches) == 0 {
		fmt.Println(
			"<pull-request> should be in either:\n" +
				" - URL Format: http[s]://*/(users|projects)/*/repos/*/pull-requests/<id>\n" +
				" - Shorthand format: [<project>/]<repo>/<id>",
		)
		os.Exit(1)
	}

	editor := os.Getenv("EDITOR")
	if args["-e"] != nil {
		editor = args["-e"].(string)
	}

	if editor == "" {
		fmt.Println(
			"Either -e or env var $EDITOR should specify edtitor to use.")
		os.Exit(1)
	}

	hostName := matches[1]
	projectName := matches[2]
	repoName := matches[5]
	pullRequestId, _ := strconv.ParseInt(matches[6], 10, 16)

	user := args["--user"].(string)
	pass := args["--pass"].(string)

	auth := gopencils.BasicAuth{user, pass}
	api := Api{hostName, auth}
	project := Project{&api, projectName}
	repo := Repo{&project, repoName}

	path := ""
	if args["<file-name>"] != nil {
		path = args["<file-name>"].(string)
	}

	pullRequest := NewPullRequest(&repo, int(pullRequestId))

	switch {
	case args["ls"]:
		showFilesList(pullRequest)
	case args["review"]:
		review(pullRequest, editor, path)
	}
}

func reviewFile(pr PullRequest, path string) (*Review, error) {
	return pr.GetReview(path)
}

func editReviewInEditor(
	editor string, reviewToEdit *Review, fileToUse *os.File,
) ([]ReviewChange, error) {
	logger.Info("writing review to file: %s", fileToUse.Name())

	AddUsageComment(reviewToEdit)
	AddVimModeline(reviewToEdit)

	WriteReview(reviewToEdit, fileToUse)

	fileToUse.Sync()

	logger.Debug("opening editor: %s %s", editor, fileToUse.Name())
	editorCmd := exec.Command(editor, fileToUse.Name())
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	err := editorCmd.Run()
	if err != nil {
		logger.Fatal(err)
	}

	fileToUse.Sync()
	fileToUse.Seek(0, os.SEEK_SET)

	logger.Debug("reading modified review back")
	editedReview, err := ReadReview(fileToUse)
	if err != nil {
		return nil, err
	}

	logger.Debug("comparing old and new reviews")
	return reviewToEdit.Compare(editedReview), nil
}

func mergeArgsWithConfig(path string) []string {
	args := make([]string, 0)

	conf, err := ioutil.ReadFile(path)

	if err != nil {
		logger.Warning("can not access config: %s", err.Error())
		return args
	}

	confLines := strings.Split(string(conf), "\n")
	for _, line := range confLines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		args = append(args, line)
	}

	fmt.Fprintf(os.Stderr, "Note: cmd line args are read from %s\n", path)

	args = append(args, os.Args[1:]...)

	return args
}

func showFilesList(pr PullRequest) {
	logger.Debug("showing list of files in PR")
	files, err := pr.GetFiles()
	if err != nil {
		logger.Error("error accessing Stash: %s", err.Error())
	}

	for _, file := range files {
		execFlag := ""
		if file.DstExec != file.SrcExec {
			if file.DstExec {
				execFlag = "-x"
			} else {
				execFlag = "+x"
			}
		}

		fmt.Printf("%2s %7s %s\n", execFlag, file.ChangeType, file.DstPath)
	}
}

func review(pr PullRequest, editor string, path string) {
	var review *Review
	var err error

	if path == "" {
		logger.Debug("downloading overview from Stash")
		review, err = pr.GetActivities()
	} else {
		logger.Debug("downloading review from Stash")
		review, err = pr.GetReview(path)
	}

	if err != nil {
		logger.Fatal(err)
	}

	if len(review.changeset.Diffs) == 0 {
		fmt.Println("Specified file is not found in pull request.")
		os.Exit(1)
	}

	tmpFile, err := ioutil.TempFile(os.TempDir(), "review.diff.")
	defer func() {
		if r := recover(); r != nil {
			printPanicMsg(r, tmpFile.Name())
		}
	}()

	changes, err := editReviewInEditor(editor, review, tmpFile)
	if err != nil {
		logger.Fatal(err)
	}

	if len(changes) == 0 {
		logger.Warning("no changes detected in review file (maybe a bug)")
		os.Exit(2)
	}

	logger.Debug("applying changes (%d)", len(changes))

	for i, change := range changes {
		fmt.Printf("(%d/%d) applying changes\n", i+1, len(changes))
		logger.Debug("change payload: %#v", change.GetPayload())
		err := pr.ApplyChange(change)
		if err != nil {
			logger.Critical("can not apply change: %s", err.Error())
		}
	}

	tmpFile.Close()
	os.Remove(tmpFile.Name())
	logger.Debug("removed tmp file: %s", tmpFile.Name())
}

func makeUrl(args map[string]interface{}) string {
	if args["--host"] == nil {
		fmt.Println("--host must be given to use shorthand syntax")
		os.Exit(1)
	}

	project := ""
	pullRequestParam := args["<pull-request>"].(string)
	if len(strings.Split(pullRequestParam, "/")) < 3 {
		if args["--project"] != nil {
			project = args["--project"].(string) + "/"
		}
	}

	projRepoPrString := project + pullRequestParam
	projRepoPr := strings.Split(projRepoPrString, "/")
	if len(projRepoPr) != 3 {
		fmt.Println(
			"--project and <pull-request> should contain 2 slashes in " +
				"sum to form <proj>/<repo>/<pull-request-id> path.")
		fmt.Printf("given: %s\n", projRepoPrString)
		os.Exit(1)
	}
	ns := "projects"
	if projRepoPr[0][0] == '~' {
		ns = "users"
		projRepoPr[0] = projRepoPr[0][1:]
	}
	tplValue := struct {
		Ns   string
		Proj string
		Repo string
		Pr   string
	}{
		ns,
		projRepoPr[0],
		projRepoPr[1],
		projRepoPr[2],
	}
	tpl := template.Must(template.New("url").Parse(args["--url"].(string)))
	url, err := tplutil.ExecuteToString(tpl, tplValue)
	logger.Debug("%#v", url)
	if err != nil {
		fmt.Println("Error forming url: %s", err)
		os.Exit(1)
	}

	return strings.TrimSuffix(args["--host"].(string), "/") + url
}

func (p CmdLineArgs) Redacted() interface{} {
	rePassFlag := regexp.MustCompile(`(\s(-p|--pass)[\s=])([^ ]+)`)
	matches := rePassFlag.FindStringSubmatch(string(p))
	if len(matches) == 0 {
		return string(p)
	} else {
		return rePassFlag.ReplaceAllString(
			string(p),
			`$1`+logging.Redact(string(matches[3])))
	}
}

func printPanicMsg(r interface{}, reviewFileName string) {
	fmt.Println()
	fmt.Println("PANIC:", r)
	fmt.Println()
	fmt.Println(string(debug.Stack()))
	fmt.Println("Well, program is crashed. This is a bug.")
	fmt.Println()
	fmt.Printf("All data you've entered are kept in the file:\n\t%s",
		reviewFileName)
	fmt.Println()
	fmt.Println()
	fmt.Printf("Feel free to open issue or PR on the:\n\t%s",
		"https://github.com/seletskiy/ash")
	fmt.Println()
}
