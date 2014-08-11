package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/bndr/gopencils"
)

func main() {
	auth := gopencils.BasicAuth{os.Args[1], os.Args[2]}
	api := Api{"git.rn", auth}
	project := Project{&api, "OAPP"}
	repo := Repo{&project, "deployer"}

	pullRequest := NewPullRequest(&repo, 1)
	diffs, err := pullRequest.GetDiffs("libdeploy/conf.go")
	if err != nil {
		fmt.Println(err)
	}

	editedReview, err := ioutil.TempFile(os.TempDir(), "review")
	if err != nil {
		log.Fatal(err)
	}

	editedReview.WriteString(diffs.String())
	editedReview.Close()

	editorCmd := exec.Command("vim", editedReview.Name())
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr
	err = editorCmd.Run()
	if err != nil {
		log.Fatal(err)
	}

	initialReview, err := ioutil.TempFile(os.TempDir(), "review")
	if err != nil {
		log.Fatal(err)
	}

	initialReview.WriteString(diffs.String())
	initialReview.Close()

	diffCmd := exec.Command("diff", "-u", "-F", "^@@", editedReview.Name(), initialReview.Name())

	diffPipe, _ := diffCmd.StdoutPipe()
	diffCmd.Start()
	diff, _ := ioutil.ReadAll(diffPipe)
	diffCmd.Wait()
	fmt.Println(string(diff))

	//diff, err := diffCmd.CombinedOutput()
	//reviewDiff, _ := diffCmd.CombinedOutput()
}
