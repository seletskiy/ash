package main

import (
	"net/url"
)

func (api *Api) GetInbox(role string) ([]PullRequest, error) {
	logger.Debug(
		"requesting pull requests count from Stash for role '%s'...",
		role,
	)

	cookies, err := api.authViaWeb()
	if err != nil {
		return nil, err
	}

	hostURL, _ := url.Parse(api.URL)
	resource := api.GetResource().Res("inbox/latest")
	resource.Api.Cookies.SetCookies(hostURL, cookies)

	prReply := struct {
		Values []PullRequest
	}{}

	err = api.DoGet(resource.Res("pull-requests", &prReply),
		map[string]string{
			"limit": "1000",
			"role":  role,
		})

	if err != nil {
		return nil, err
	}

	return prReply.Values, nil
}
