package main

import (
	"fmt"

	"github.com/bndr/gopencils"
)

type Repo struct {
	*Project
	Name     string
	Resource *gopencils.Resource
}

func (repo *Repo) GetPullRequest(id int64) PullRequest {
	return PullRequest{
		Repo:     repo,
		Id:       id,
		Resource: repo.Resource.Res("pull-requests").Id(fmt.Sprint(id)),
	}
}

func (repo *Repo) ListPullRequest(state string) ([]PullRequest, error) {
	reply := struct {
		Size       int
		Limit      int
		IsLastPage bool
		Values     []PullRequest
	}{}

	query := map[string]string{
		"state": state,
	}

	err := repo.DoGet(repo.Resource.Res("pull-requests", &reply), query)
	if err != nil {
		return nil, err
	}

	return reply.Values, nil
}
