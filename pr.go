package main

import (
	"fmt"

	"github.com/bndr/gopencils"
	"github.com/seletskiy/godiff"
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

func (pr *PullRequest) GetReview(path string) (*Review, error) {
	review := &Review{godiff.Changeset{}}

	_, err := pr.Resource.Res("diff").Id(path, &review.changeset).Get()
	if err != nil {
		return nil, err
	}

	review.changeset.ForEachLine(func(diff *godiff.Diff, line *godiff.Line) {
		for _, id := range line.CommentIds {
			for _, c := range diff.LineComments {
				if c.Id == id {
					line.Comments = append(line.Comments, c)
				}
			}
		}
	})

	review.changeset.Path = path

	return review, nil
}

//func (pr *PullRequest) ApplyChange(change ReviewChange) error {
//    switch change.Change {
//    case CommentAdded:
//    }
//    return nil
//}
