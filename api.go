package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"time"

	"github.com/bndr/gopencils"
)

type Api struct {
	Host        string
	Auth        gopencils.BasicAuth
	AuthCookies []*http.Cookie
}

type Project struct {
	*Api
	Name string
}

type ApiError struct {
	Errors []struct {
		Message string
	}
}

type UnixTimestamp int

func (u UnixTimestamp) String() string {
	return time.Unix(int64(u/1000), 0).Format("Mon Jan _2 15:04 2006")
}

func (api Api) GetResource() *gopencils.Resource {
	return gopencils.Api(fmt.Sprintf("%s/rest", api.Host), &api.Auth)
}

func (api Api) authViaWeb() ([]*http.Cookie, error) {
	if api.AuthCookies != nil {
		return api.AuthCookies, nil
	}

	jar, _ := cookiejar.New(nil)
	client := http.Client{Jar: jar}

	_, err := client.PostForm(api.Host+"j_stash_security_check",
		url.Values{
			"j_username": {api.Auth.Username},
			"j_password": {api.Auth.Password},
		})

	if err != nil {
		return nil, err
	}

	hostUrl, _ := url.Parse(api.Host)
	api.AuthCookies = jar.Cookies(hostUrl)

	return api.AuthCookies, nil
}

func (api Api) DoGet(
	res *gopencils.Resource,
	payload ...interface{},
) error {
	logger.Debug("performing GET %s %v", res.Url, payload)
	return api.doRequest(res,
		func() (*gopencils.Resource, error) { return res.Get(payload...) })
}

func (api Api) DoPost(
	res *gopencils.Resource,
	payload ...interface{},
) error {
	logger.Debug("performing POST %s %v", res.Url, payload)
	return api.doRequest(res,
		func() (*gopencils.Resource, error) { return res.Post(payload...) })
}

func (api Api) DoPut(
	res *gopencils.Resource,
	payload ...interface{},
) error {
	logger.Debug("performing PUT %s %v", res.Url, payload)
	return api.doRequest(res,
		func() (*gopencils.Resource, error) { return res.Put(payload...) })
}

func (api Api) DoDelete(
	res *gopencils.Resource,
	payload ...interface{},
) error {
	logger.Debug("performing DELETE %s %v", res.Url, payload)
	return api.doRequest(res,
		func() (*gopencils.Resource, error) { return res.Delete(payload...) })
}

func (api Api) doRequest(
	res *gopencils.Resource,
	doFunc func() (*gopencils.Resource, error),
) error {
	resp, err := doFunc()
	if err != nil {
		return err
	}

	if err := checkErrorStatus(resp); err != nil {
		logger.Warning("Stash returned error code: %d", resp.Raw.StatusCode)
		return err
	} else {
		logger.Debug("Stash returned status code: %d", resp.Raw.StatusCode)
	}

	return nil
}

func (project Project) GetRepo(name string) Repo {
	return Repo{
		Project: &project,
		Name:    name,
		Resource: project.GetResource().
			Res("api/1.0").Res(project.Name).Res("repos").Res(name),
	}
}

func checkErrorStatus(resp *gopencils.Resource) error {
	switch resp.Raw.StatusCode {
	case 200, 201, 204:
		return nil

	case 400, 401, 404, 409:
		errorBody, _ := ioutil.ReadAll(resp.Raw.Body)
		if len(errorBody) > 0 {
			return stashApiError(errorBody)
		} else {
			return unexpectedStatusCode(resp.Raw.StatusCode)
		}

	default:
		return unexpectedStatusCode(resp.Raw.StatusCode)
	}
}
