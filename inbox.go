package main

import (
	"fmt"
	"net/url"
)

func (api *Api) GetInbox() ([]PullRequest, error) {
	logger.Debug("requesting pull requests count from Stash...")

	cookies, err := api.authViaWeb()
	if err != nil {
		return nil, err
	}

	hostUrl, _ := url.Parse(api.Host)
	resource := api.GetResource().Res("inbox/latest")
	resource.Api.Cookies.SetCookies(hostUrl, cookies)

	countReply := struct {
		Count int
	}{}

	err = api.DoGet(resource.Res("pull-requests/count", &countReply))
	if err != nil {
		return nil, err
	}

	logger.Debug("Stash returned %d amount of pull requests", countReply.Count)

	prReply := struct {
		Values []PullRequest
	}{}

	err = api.DoGet(resource.Res("pull-requests", &prReply),
		map[string]string{
			"limit": fmt.Sprint(countReply.Count),
		})
	if err != nil {
		return nil, err
	}

	return prReply.Values, nil
}
