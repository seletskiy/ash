package main

import "github.com/bndr/gopencils"

type Api struct {
	Host string
	Auth gopencils.BasicAuth
}

type Project struct {
	*Api
	Name string
}

type Repo struct {
	*Project
	Name string
}

type ApiError struct {
	Errors []struct {
		Message string
	}
}
