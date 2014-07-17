package main

import (
	"fmt"
	"os"

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

	fmt.Println(*diffs)
}
