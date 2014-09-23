package main

import (
	"io/ioutil"
	"time"

	"github.com/bndr/gopencils"
)

type Api struct {
	Host string
	Auth gopencils.BasicAuth
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
	return time.Unix(int64(u/1000), 0).Format(time.ANSIC)
}

func checkErrorStatus(resp *gopencils.Resource) error {
	logger.Debug("Stash replied with HTTP code: %d", resp.Raw.StatusCode)

	switch resp.Raw.StatusCode {
	case 200:
		fallthrough
	case 201:
		fallthrough
	case 204:
		return nil
	case 400:
		fallthrough
	case 401:
		fallthrough
	case 404:
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
