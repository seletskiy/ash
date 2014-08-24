package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/bndr/gopencils"
	"github.com/docopt/docopt-go"
)

var (
	reStashURL = regexp.MustCompile(
		`https?://(.*)/` +
			`((users|projects)/([^/]+))` +
			`/repos/([^/]+)` +
			`/pull-requests/(\d+)`)
)

var configPath = os.Getenv("HOME") + "/.config/ash/ashrc"

func parseCmdLine(cmd []string) (map[string]interface{}, error) {
	help := `Atlassian Stash Reviewer.

Most convient usage is specify pull request url and file you want to review:
  ash review http://stash.local/projects/.../repos/.../pull-requests/... file

Ash then open $EDITOR for commenting on pull request.

You can add comments by just specifying them after line you want to comment,
beginning with '# '.

You can delete comment by deleting it from file, and, of course, modify comment
you own by modifying it in the file.

After you finish your edits, just save file and exit from editor. Ash will
apply all changes made to the review.

Usage:
  ash [options] review <pull-request-url> <file-name>
  ash -h | --help

Options:
  -h --help           Show this help.
  -u --user=<user>    Stash username.
  -p --pass=<pass>    Stash password.
`

	args, err := docopt.Parse(help, cmd, true, "0.1 beta", false)
	return args, err
}

func main() {
	rawArgs := mergeArgsWithConfig(configPath)

	args, _ := parseCmdLine(rawArgs)

	url := args["<pull-request-url>"].(string)

	matches := reStashURL.FindStringSubmatch(url)
	if len(matches) == 0 {
		fmt.Println(
			"<pull-request-url> should be in following format:\n",
			"http[s]://stash.local/users|projects/*/repos/*/pull-requests/*")
		os.Exit(1)
	}

	hostName := matches[1]
	projectName := matches[2]
	repoName := matches[5]
	pullRequestId, _ := strconv.ParseInt(matches[6], 10, 16)

	if args["--user"] == nil || args["--pass"] == nil {
		fmt.Println(
			"--user and --pass should be specified.")
		os.Exit(1)
	}

	user := args["--user"].(string)
	pass := args["--pass"].(string)

	auth := gopencils.BasicAuth{user, pass}
	api := Api{hostName, auth}
	project := Project{&api, projectName}
	repo := Repo{&project, repoName}

	path := args["<file-name>"].(string)

	pullRequest := NewPullRequest(&repo, int(pullRequestId))
	initialReview, err := pullRequest.GetReview(path)
	if err != nil {
		fmt.Println(err)
	}

	if len(initialReview.changeset.Diffs) == 0 {
		fmt.Println("Specified file is not found in pull request.")
		os.Exit(1)
	}

	modifiedReviewFile, err := ioutil.TempFile(os.TempDir(), "review")
	if err != nil {
		log.Fatal(err)
	}

	modifiedReviewFile.WriteString(initialReview.String())
	modifiedReviewFile.Sync()

	editorCmd := exec.Command(os.Getenv("EDITOR"), modifiedReviewFile.Name())
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	err = editorCmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	modifiedReviewFile.Close()

	modifiedReview, _ := ParseReviewFile(modifiedReviewFile.Name())

	changes := initialReview.Compare(modifiedReview)
	for _, change := range changes {
		pullRequest.ApplyChange(change)
	}
}

func mergeArgsWithConfig(path string) []string {
	args := make([]string, 0)
	args = append(args, os.Args[1:]...)

	conf, err := ioutil.ReadFile(path)

	if err != nil {
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

	return args
}
