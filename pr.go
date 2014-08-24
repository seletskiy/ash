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
			"http://%s/rest/api/1.0/%s/repos/%s/pull-requests/%d",
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

func (pr *PullRequest) ApplyChange(change ReviewChange) error {
	if _, ok := change["id"]; ok {
		if _, ok := change["text"]; ok {
			return pr.modifyComment(change)
		} else {
			return pr.removeComment(change)
		}
	} else {
		return pr.addComment(change)
	}

	panic(fmt.Sprintf("unexpected <change> argument: %s", change))
	return nil
}

func (pr *PullRequest) addComment(change ReviewChange) error {
	resp := godiff.Comment{}

	_, err := pr.Resource.Res("comments", &resp).Post(change)

	if resp.Id > 0 {
		fmt.Printf("[debug] comment added: %d\n", resp.Id)
	} else {
		fmt.Printf("[debug] fail to add comment:\n%s\n", change)
	}

	return err
}

func (pr *PullRequest) modifyComment(change ReviewChange) error {
	query := map[string]string{
		"version": fmt.Sprint(change["version"]),
	}

	resp := godiff.Comment{}
	_, err := pr.Resource.Res("comments").
		Id(fmt.Sprint(change["id"]), &resp).
		SetQuery(query).Put(change)

	if resp.Id > 0 {
		fmt.Printf("[debug] comment modified: %d\n", resp.Id)
	} else {
		fmt.Printf("[debug] fail to modify comment:\n%s\n", change)
	}

	return err
}

func (pr *PullRequest) removeComment(change ReviewChange) error {
	query := map[string]string{
		"version": fmt.Sprint(change["version"]),
	}

	resp := make(map[string]interface{})

	_, err := pr.Resource.Res("comments").Id(fmt.Sprint(change["id"]), &resp).
		SetQuery(query).Delete()

	fmt.Printf("[debug] comment wasted: %d\n", change["id"])

	return err
}
