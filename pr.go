package main

import (
	"fmt"

	"github.com/bndr/gopencils"
)

type PullRequest struct {
	*Repo
	Id       int
	Resource *gopencils.Resource
}

func NewPullRequest(repo *Repo, id int) PullRequest {
	return PullRequest{
		Repo: repo,
		Id:   id,
		Resource: gopencils.Api(fmt.Sprintf(
			"http://%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d",
			repo.Host,
			repo.Project.Name,
			repo.Name,
			id,
		), &repo.Auth),
	}
}

func (pr *PullRequest) GetDiffs(path string) (*Diffs, error) {
	diffs := Diffs{}

	_, err := pr.Resource.Res("diff").Id(path, &diffs).Get()
	if err != nil {
		return nil, err
	}

	diffs.ForEachLines(func(diff *Diff, line *Line) {
		for _, id := range line.CommentIds {
			for _, c := range (*diff).LineComments {
				if c.Id == id {
					line.Comments = append(line.Comments, c)
				}
			}
		}
	})

	return &diffs, nil
}
