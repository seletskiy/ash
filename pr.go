package main

import (
	"fmt"
	"io/ioutil"

	"github.com/bndr/gopencils"
	"github.com/seletskiy/godiff"
)

type unexpectedStatusCode int

func (u unexpectedStatusCode) Error() string {
	return fmt.Sprintf("unexpected status code from Stash: %d", u)
}

type stashApiError []byte

func (s stashApiError) Error() string {
	return string(s)
}

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
	result := godiff.Changeset{}

	resp, err := pr.Resource.Res("diff").Id(path, &result).Get()
	if err != nil {
		return nil, err
	}

	status := resp.Raw.StatusCode
	switch status {
	case 200:
		// ok
	case 400:
		fallthrough
	case 401:
		fallthrough
	case 404:
		errorBody, _ := ioutil.ReadAll(resp.Raw.Body)
		if len(errorBody) > 0 {
			return nil, stashApiError(errorBody)
		} else {
			return nil, unexpectedStatusCode(status)
		}
	default:
		return nil, unexpectedStatusCode(status)
	}

	// TODO: refactor block
	result.ForEachLine(
		func(diff *godiff.Diff, _ *godiff.Hunk, _ *godiff.Segment, line *godiff.Line) {
			for _, id := range line.CommentIds {
				for _, c := range diff.LineComments {
					if c.Id == id {
						line.Comments = append(line.Comments, c)
					}
				}
			}
		})

	result.Path = path

	return &Review{result}, nil
}

func (pr *PullRequest) GetActivities() (ReviewActivities, error) {
	query := map[string]string{
		"limit": "25",
	}

	activities := ReviewActivities{}
	_, err := pr.Resource.Res("activities", &activities).Get(query)
	if err != nil {
		return nil, err
	}

	return activities, nil
}

func (pr *PullRequest) ApplyChange(change ReviewChange) error {
	switch c := change.(type) {
	case ReplyAdded:
		pr.addComment(c)
	case LineCommentAdded:
		pr.addComment(c)
	case CommentRemoved:
		pr.removeComment(c)
	case CommentModified:
		pr.modifyComment(c)
	default:
		panic(fmt.Sprintf("unexpected <change> argument: %s", change))
	}

	return nil
}

func (pr *PullRequest) addComment(change ReviewChange) error {
	result := godiff.Comment{}
	resp, err := pr.Resource.Res("comments", &result).Post(change.GetPayload())

	if err != nil {
		return err
	}

	status := resp.Raw.StatusCode

	//apiErr := ApiError{}
	if status == 400 || status == 401 || status == 404 {
		fmt.Println(status)
	}

	if result.Id > 0 {
		fmt.Printf("[debug] comment added: %d\n", result.Id)
	} else {
		fmt.Printf("[debug] fail to add comment:\n%s\n", change)
	}

	return nil
}

func (pr *PullRequest) modifyComment(change CommentModified) error {
	query := map[string]string{
		"version": fmt.Sprint(change.comment.Version),
	}

	result := godiff.Comment{}
	_, err := pr.Resource.
		Res("comments").
		Id(fmt.Sprint(change.comment.Id), &result).
		SetQuery(query).
		Put(change)

	if result.Id > 0 {
		fmt.Printf("[debug] comment modified: %d\n", result.Id)
	} else {
		fmt.Printf("[debug] fail to modify comment:\n%s\n", change)
	}

	return err
}

func (pr *PullRequest) removeComment(change CommentRemoved) error {
	query := map[string]string{
		"version": fmt.Sprint(change.comment.Version),
	}

	result := make(map[string]interface{})
	_, err := pr.Resource.
		Res("comments").
		Id(fmt.Sprint(change.comment.Id), &result).
		SetQuery(query).
		Delete()

	fmt.Printf("[debug] comment wasted: %d\n", change.comment.Id)

	return err
}
