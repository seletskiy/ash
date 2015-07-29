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

	"github.com/bndr/gopencils"
	"github.com/docopt/docopt-go"
	"github.com/op/go-logging"
)

var (
	reStashURL = regexp.MustCompile(
		`(https?://.*/)` +
			`((users|projects)/([^/]+))` +
			`/repos/([^/]+)` +
			`/pull-requests/(\d+)`)
)

var configPath = os.Getenv("HOME") + "/.config/ash/ashrc"

var logger = logging.MustGetLogger("main")

var tmpWorkDir = ""
var panicState = false

const logFormat = "%{time:15:04:05.00} [%{level:.4s}] %{message}"
const logFormatColor = "%{color}" + logFormat + "%{color:reset}"

const startUrlExample = "http[s]://<host>/(users|projects)/<project>/repos/<repo>/pull-requests/<id>"

type CmdLineArgs string

func parseCmdLine(cmd []string) (map[string]interface{}, error) {
	help := `Atlassian Stash Reviewer.

Most convenient usage is specify pull request url and file you want to review:
  ash ` + startUrlExample + ` review <file-to-review>

However, you can set up --url and --project flags in ~/.config/ash/ashrc file
and access pull requests by shorthand commands:
  ash proj/mycoolrepo/1 review  # if --url is given
  ash mycoolrepo/1 review       # if --url and --project is given
  ash mycoolrepo ls-reviews     # --//--

Ash then open $EDITOR for commenting on pull request.

You can add comments by just specifying them after line you want to comment,
beginning with '# '.

You can delete comment by deleting it from file, and, of course, modify comment
you own by modifying it in the file.

After you finish your edits, just save file and exit from editor. Ash will
apply all changes made to the review.

If <file-name> is omitted, ash welcomes you to review the overview.

'ls' command can be used to list various things, including:
* files in pull request;
* opened/merged/declined pull requests for repo;
* repositories in specified project [NOT IMPLEMENTED];
* projects [NOT IMPLEMENTED];

Usage:
  ash [options] <project>/<repo>/<pr> [review] [<file-name>] [-w]
  ash [options] <project>/<repo>/<pr> ls
  ash [options] <project>/<repo>/<pr> (approve|decline|merge)
  ash [options] <project>/<repo> ls-reviews [-d] [(open|merged|declined)]
  ash [options] inbox [-d] [(reviewer|author|all)]
  ash -h | --help
  ash -v | --version

Options:
  -h --help          Show this help.
  -v --version       Show version
  -u --user=<user>   Stash username.
  -p --pass=<pass>   Stash password. You want to set this flag in .ashrc file.
  -d                 Show descriptions for the listed PRs.
  -l=<count>         Number of activities to retrieve. [default: 1000]
  -w                 Ignore whitespaces
  -e=<editor>        Editor to use. This has priority over $EDITOR env var.
  --debug=<level>    Verbosity [default: 0].
  --url=<url>        Stash server URL.  http:// will be used if no protocol is
                     specified.
  --input=<input>    File for loading diff in review file
  --output=<output>  Output review to specified file. Editor is ignored.
  --origin=<origin>  Do not download review from stash and use specified file
                     instead.
  --project=<proj>   Use to specify default project that can be used when
                     serching pull requests. Can be set in either <project> or
                     <project>/<repo> format.
  --no-color         Do not use color in output.
`

	args, err := docopt.Parse(help, cmd, true, "1.2", false, false)

	if _, ok := err.(*docopt.UserError); ok {
		fmt.Println()
		fmt.Println("Command line entered is invalid.")
		fmt.Println()
		fmt.Println(
			"Arguments were merged with config values and " +
				"the resulting command line is:")
		fmt.Printf("\t%s\n\n", CmdLineArgs(fmt.Sprintf("%s", cmd)).Redacted())
		os.Exit(1)
	}

	if err == nil && args == nil {
		os.Exit(0)
	}

	return args, err
}

func main() {
	rawArgs := mergeArgsWithConfig(configPath)

	args, err := parseCmdLine(rawArgs)
	if err != nil {
		logger.Critical(err.Error())
	}

	tmpWorkDir, err = ioutil.TempDir(os.TempDir(), "ash.")
	if err != nil {
		logger.Critical(err.Error())
	}

	setupLogger(args)

	logger.Info("cmd line args are read from %s", configPath)
	logger.Debug("cmd line args: %s", CmdLineArgs(fmt.Sprintf("%s", rawArgs)))

	if args["--user"] == nil || args["--pass"] == nil {
		fmt.Println("--user and --pass should be specified.")
		os.Exit(1)
	}

	uri := parseUri(args)

	if !strings.HasPrefix(uri.base, "http") {
		uri.base = "http://" + uri.base
	}

	if !strings.HasSuffix(uri.base, "/") {
		uri.base = uri.base + "/"
	}

	user := args["--user"].(string)
	pass := args["--pass"].(string)

	auth := gopencils.BasicAuth{user, pass}
	api := Api{uri.base, auth, nil}
	project := Project{&api, uri.project}
	repo := project.GetRepo(uri.repo)

	switch {
	case args["<project>/<repo>/<pr>"] != nil:
		reviewMode(args, repo, uri.pr)
	case args["<project>/<repo>"] != nil:
		repoMode(args, repo)
	case args["inbox"].(bool):
		inboxMode(args, api)
	}

	if !panicState {
		// in case of everything is fine
		logger.Debug("removing %s", tmpWorkDir)
		os.RemoveAll(tmpWorkDir)
	}
}

func setupLogger(args map[string]interface{}) {
	debugLogFile, err := os.Create(tmpWorkDir + "/debug.log")
	if err != nil {
		logger.Critical(err.Error())
	}

	debugLog := logging.AddModuleLevel(
		logging.NewLogBackend(debugLogFile, "", 0))

	debugLog.SetLevel(logging.DEBUG, "main")

	logging.SetBackend(
		logging.MultiLogger(
			debugLog,
			logging.AddModuleLevel(logging.NewLogBackend(os.Stderr, "", 0))),
	)

	targetLogFormat := logFormatColor
	if args["--no-color"].(bool) {
		targetLogFormat = logFormat
	}

	logging.SetFormatter(logging.MustStringFormatter(targetLogFormat))

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

	debugLog.SetLevel(logging.DEBUG, "main")
}

func inboxMode(args map[string]interface{}, api Api) {
	roles := []string{"author", "reviewer"}
	for _, role := range roles {
		if args[role].(bool) {
			roles = []string{role}
			break
		}
	}

	channels := make(map[string]chan []PullRequest)
	for _, role := range roles {
		channels[role] = requestInboxFor(role, api)
	}

	for _, role := range roles {
		for _, pullRequest := range <-channels[role] {
			printPullRequest(pullRequest, args["-d"].(bool), false)
		}
	}
}

func requestInboxFor(role string, api Api) chan []PullRequest {
	resultChannel := make(chan []PullRequest, 0)

	go func() {
		reviews, err := api.GetInbox(role)
		if err != nil {
			logger.Critical(
				"error retrieving inbox for '%s': %s",
				role,
				err.Error(),
			)
		}

		resultChannel <- reviews
	}()

	return resultChannel
}

func reviewMode(args map[string]interface{}, repo Repo, pr int64) {
	editor := os.Getenv("EDITOR")
	if args["-e"] != nil {
		editor = args["-e"].(string)
	}

	path := ""
	if args["<file-name>"] != nil {
		path = args["<file-name>"].(string)
	}

	input := ""
	if args["--input"] != nil {
		input = args["--input"].(string)
	}

	output := ""
	if args["--output"] != nil {
		output = args["--output"].(string)
	}

	ignoreWhitespaces := false
	if args["-w"].(bool) {
		ignoreWhitespaces = true
	}

	activitiesLimit := args["-l"].(string)

	pullRequest := repo.GetPullRequest(pr)

	origin := ""
	if args["--origin"] != nil {
		origin = args["--origin"].(string)
	}

	switch {
	case args["ls"]:
		showFilesList(pullRequest)
	case args["review"]:
		review(
			pullRequest, editor, path,
			origin, input, output,
			activitiesLimit, ignoreWhitespaces,
		)

	case args["approve"].(bool):
		approve(pullRequest)
	case args["decline"].(bool):
		decline(pullRequest)
	case args["merge"].(bool):
		merge(pullRequest)
	}
}

func approve(pr PullRequest) {
	logger.Debug("Approving pr")
	err := pr.Approve()
	if err != nil {
		logger.Critical("error approving: %s", err.Error())
		os.Exit(1)
	}

	fmt.Println("Pull request successfully approved")
}

func decline(pr PullRequest) {
	logger.Debug("Declining pr")
	err := pr.Decline()
	if err != nil {
		logger.Critical("error declining: %s", err.Error())
		os.Exit(1)
	}

	fmt.Println("Pull request successfully declined")
}

func merge(pr PullRequest) {
	logger.Debug("Merging pr")
	err := pr.Merge()
	if err != nil {
		logger.Critical("error merging: %s", err.Error())
		os.Exit(1)
	}

	fmt.Println("Pull request successfully merged")
}

func repoMode(args map[string]interface{}, repo Repo) {
	switch {
	case args["ls-reviews"]:
		state := "open"
		switch {
		case args["declined"]:
			state = "declined"
		case args["merged"]:
			state = "merged"
		}
		showReviewsInRepo(repo, state, args["-d"].(bool))
	}
}

func showReviewsInRepo(repo Repo, state string, withDesc bool) {
	reviews, err := repo.ListPullRequest(state)

	if err != nil {
		logger.Critical("can not list reviews: %s", err.Error())
	}

	for _, r := range reviews {
		printPullRequest(r, withDesc, true)
	}
}

func printPullRequest(pr PullRequest, withDesc bool, printStatus bool) {
	textId := fmt.Sprintf("%s/%s/%d ",
		strings.ToLower(pr.FromRef.Repository.Project.Key),
		pr.FromRef.Repository.Slug,
		pr.Id,
	)

	fmt.Printf("%-30s ", textId)

	if len(pr.Attributes.CommentCount) != 0 {
		fmt.Printf("(%3s) ", pr.Attributes.CommentCount[0])
	} else {
		fmt.Printf("(  0) ")
	}

	if printStatus {
		fmt.Printf("%s ", pr.State)
	}

	fmt.Printf(
		"[%6s] %s: ",
		pr.UpdatedDate,
		pr.Author.User.DisplayName,
	)

	refSegments := strings.Split(pr.FromRef.Id, "/")
	branchName := refSegments[len(refSegments)-1]
	fmt.Printf("%s", branchName)

	if withDesc && pr.Description != "" {
		desc := fmt.Sprintf("\n---\n%s\n---", pr.Description)
		fmt.Println(desc)
	}

	fmt.Println()

}

func parseUri(args map[string]interface{}) (
	result struct {
		base    string
		project string
		repo    string
		pr      int64
	},
) {
	uri := ""
	keyName := ""
	should := 0

	if args["<project>/<repo>/<pr>"] != nil {
		keyName = "<project>/<repo>/<pr>"
		uri = args[keyName].(string)
		should = 3
	}

	if args["<project>/<repo>"] != nil {
		keyName = "<project>/<repo>"
		uri = args[keyName].(string)
		should = 2
	}

	matches := reStashURL.FindStringSubmatch(uri)
	if len(matches) != 0 {
		result.base = matches[1]
		result.project = matches[2]
		result.repo = matches[5]
		result.pr, _ = strconv.ParseInt(matches[6], 10, 16)

		return result
	}

	if args["--url"] == nil {
		fmt.Println(
			"In case of shorthand syntax --url should be specified")
		os.Exit(1)
	}

	if should == 0 {
		result.base = args["--url"].(string)
		return
	}

	matches = strings.Split(uri, "/")

	result.base = args["--url"].(string)

	if len(matches) == 2 && should == 3 && args["--project"] != nil {
		result.repo = matches[0]
		result.pr, _ = strconv.ParseInt(matches[1], 10, 16)
	}

	if args["--project"] != nil {
		result.project = args["--project"].(string)
	}

	if len(matches) == 1 && should == 2 {
		result.repo = matches[0]
	}

	if len(matches) == 2 && should == 2 {
		result.project = matches[0]
		result.repo = matches[1]
	}

	if len(matches) >= 3 && should == 3 {
		result.project = matches[0]
		result.repo = matches[1]
		result.pr, _ = strconv.ParseInt(matches[2], 10, 16)
	}

	enough := result.project != "" && result.repo != "" &&
		(result.pr != 0 || should == 2)

	if !enough {
		fmt.Println(
			"<pull-request> should be in either:\n" +
				" - URL Format: " + startUrlExample + "\n" +
				" - Shorthand format: " + keyName,
		)
		os.Exit(1)
	}

	if result.project[0] == '~' || result.project[0] == '%' {
		result.project = "users/" + result.project[1:]
	} else {
		result.project = "projects/" + result.project
	}

	return result
}

func editReviewInEditor(
	editor string, reviewToEdit *Review, fileToUse *os.File,
) ([]ReviewChange, error) {
	if editor == "" {
		fileToUse.Close()

		fmt.Printf("%s", fileToUse.Name())
		os.Exit(0)
	}

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
	} else {
		confLines := strings.Split(string(conf), "\n")
		for _, line := range confLines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			args = append(args, line)
		}
	}

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
				execFlag = " +x"
			} else {
				execFlag = " -x"
			}
		}

		fmt.Printf("%7s %s%s\n", file.ChangeType, file.DstPath, execFlag)
	}
}

func review(
	pr PullRequest, editor string,
	path string,
	origin string, input string, output string,
	activitiesLimit string,
	ignoreWhitespaces bool,
) {
	var review *Review
	var err error

	if origin == "" {
		if path == "" {
			logger.Debug("downloading overview from Stash")
			review, err = pr.GetActivities(activitiesLimit)
		} else {
			logger.Debug("downloading review from Stash")
			review, err = pr.GetReview(path, ignoreWhitespaces)
		}

		if len(review.changeset.Diffs) == 0 {
			fmt.Println("Specified file is not found in pull request.")
			os.Exit(1)
		}
	} else {
		logger.Debug("using origin review from file %s", origin)
		originFile, err := os.Open(origin)
		if err != nil {
			logger.Fatal(err)
		}

		defer originFile.Close()

		review, err = ReadReview(originFile)
		if err != nil {
			logger.Fatal(err)
		}

		if path == "" {
			review.isOverview = true
		}
	}

	if err != nil {
		logger.Fatal(err)
	}

	var changes []ReviewChange
	var fileToUse *os.File

	defer func() {
		if r := recover(); r != nil {
			panicState = true
			printPanicMsg(r, fileToUse.Name())
		}
	}()

	if input != "" {
		logger.Debug("reading review from file %s", input)

		fileToUse, err = os.Open(input)
		if err != nil {
			logger.Fatal(err)
		}

		editedReview, err := ReadReview(fileToUse)

		if err != nil {
			logger.Fatal(err)
		}

		logger.Debug("comparing old and new reviews")
		changes = review.Compare(editedReview)
	} else {
		pullRequestInfo, err := pr.GetInfo()
		if err != nil {
			fmt.Println("Error while obtaining pull request info: %s", err)
			os.Exit(1)
		}

		printFileName := false
		writeAndExit := false

		if output == "" {
			output = tmpWorkDir + "/review.diff"
		} else if output == "-" {
			writeAndExit = true
			printFileName = true
			output = tmpWorkDir + "/review.diff"
		} else {
			writeAndExit = true
		}

		fileToUse, err := WriteReviewToFile(
			pullRequestInfo.Links.Self[0].Href, review, output,
		)

		if err != nil {
			logger.Fatal(err)
		}

		if writeAndExit {
			if printFileName {
				fmt.Println(output)
			}

			os.Exit(0)
		}

		changes, err = editReviewInEditor(editor, review, fileToUse)
		if err != nil {
			logger.Fatal(err)
		}
	}

	if len(changes) == 0 {
		logger.Info("no changes detected in review file (maybe a bug)")
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
}

func WriteReviewToFile(
	url string, review *Review, output string,
) (
	*os.File, error,
) {
	fileToUse, err := os.Create(output)
	if err != nil {
		return nil, err
	}

	logger.Info("writing review to file: %s", fileToUse.Name())

	AddAshModeline(url, review)

	AddUsageComment(review)

	WriteReview(review, fileToUse)

	fileToUse.Sync()

	return fileToUse, nil
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
	fmt.Println("Well, program has crashed. This is a bug.")
	fmt.Println()
	fmt.Printf("All data you've entered are kept in the file:\n\t%s",
		reviewFileName)
	fmt.Println()
	fmt.Printf("Debug log of program execution can be found at:\n\t%s",
		tmpWorkDir+"/debug.log")
	fmt.Println()
	fmt.Printf("Feel free to open issue or PR on the:\n\t%s",
		"https://github.com/seletskiy/ash")
	fmt.Println()
}
