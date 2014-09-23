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

func (p Project) GetRepo(name string) Repo {
	return Repo{
		Name: name,
		Resource: gopencils.Api(fmt.Sprintf(
			"%s/rest/api/1.0/%s/repos/%s",
			p.Host,
			p.Name,
			name),
			&p.Auth),
	}
}

func (repo *Repo) GetPullRequest(id int64) PullRequest {
	return PullRequest{
		Repo:     repo,
		Id:       id,
		Resource: repo.Resource.Res("pull-requests").Id(fmt.Sprintf("%d", id)),
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

	resp, err := repo.Resource.Res("pull-requests", &reply).Get(query)

	if err != nil {
		return nil, err
	}

	if err := checkErrorStatus(resp); err != nil {
		return nil, err
	}

	return reply.Values, nil
}
