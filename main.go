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
	project := Project{&api, "users/s.seletskiy"}
	repo := Repo{&project, "lecture-go"}

	path := "chat/main.go"

	pullRequest := NewPullRequest(&repo, 1)
	initialReview, err := pullRequest.GetReview(path)
	if err != nil {
		fmt.Println(err)
	}

	modifiedReviewFile, err := ioutil.TempFile(os.TempDir(), "review")
	if err != nil {
		log.Fatal(err)
	}

	modifiedReviewFile.WriteString(initialReview.String())
	modifiedReviewFile.Sync()

	editorCmd := exec.Command("vim", modifiedReviewFile.Name())
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
