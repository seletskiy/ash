package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	"github.com/bndr/gopencils"
	"github.com/seletskiy/godiff"
)

func main() {
	auth := gopencils.BasicAuth{os.Args[1], os.Args[2]}
	api := Api{"git.rn", auth}
	project := Project{&api, "OAPP"}
	repo := Repo{&project, "deployer"}

	pullRequest := NewPullRequest(&repo, 1)
	initialReview, err := pullRequest.GetReview("libdeploy/conf.go")
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

	modifiedReviewFile.Seek(0, os.SEEK_SET)
	modifiedReview, _ := godiff.ParseDiff(modifiedReviewFile)

	fmt.Println(modifiedReview)

	//changes := initialReview.Compare(modifiedReview)
	//for _, change := range changes {
	//    switch change.Change {
	//    case CommentAdded:

	//    }
	//}
}
