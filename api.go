package gash

import (
	"fmt"
	"strings"

	"github.com/bndr/gopencils"
)

type Diffs struct {
	Whitespace string
	Diffs      []Diff
}

type Diff struct {
	Truncated bool
	Source    struct {
		Parent string
		Name   string
	}
	Destination struct {
		Parent string
		Name   string
	}
	Hunks        []Hunk
	LineComments []Comment
}

type Hunk struct {
	SourceLine      int
	SourceSpan      int
	DestinationLine int
	DestinationSpan int
	Truncated       string
	Segments        []Segment
}

type Segment struct {
	Type      string
	Truncated bool
	Lines     []Line
}

type Line struct {
	Destination    int
	Source         int
	Line           string
	Truncated      bool
	ConflictMarker string
	CommentIds     []int
	Comments       []Comment
}

type Comment struct {
	Id          int
	Version     int
	Text        string
	CreatedDate int
	UpdatedDate int
	Comment     []Comment
	Author      struct {
		Name         string
		EmailAddress string
		Id           int
		DisplayName  string
		Active       bool
		Slug         string
		Type         string
	}
	Anchor struct {
		FromHash string
		ToHash   string
		Line     int
		LineType string
	}
	PermittedOperations struct {
		Editable  bool
		Deletable bool
	}
}

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
		), repo.Auth),
	}
}

func (pr *PullRequest) GetDiffs(path string) (*Diffs, error) {
	diffs := Diffs{}

	_, err := pr.Resource.Res("diff").Id(path, diffs).Get()
	if err != nil {
		return nil, err
	}

	diffs.ForEachLines(func(diff *Diff, line *Line) {
		for _, id := range line.CommentIds {
			for _, c := range (*diff).LineComments {
				if c.Id == id {
					line.Comments = append(line.Comments, c)
				}
			}
		}
	})

	return &diffs, nil
}

func (d Diffs) ForEachLines(callback func(*Diff, *Line)) {
	for _, diff := range d.Diffs {
		for _, hunk := range diff.Hunks {
			for _, segment := range hunk.Segments {
				for _, line := range segment.Lines {
					callback(&diff, &line)
				}
			}
		}
	}
}

func (d Diff) String() string {
	res := make(string{}, len(d.Hunks))
	for i, hunk := range d.Hunks {
		res[i] = hunk.String()
	}

	return strings.Join(res, "\n")
}

func (h Hunk) String() string {
	res := make(string{}, len(h.Segments))
	for i, segment := range h.Segments {
		res[i] = segment.String()
	}

	return strings.Join(res, "\n")
}

func (s Segment) String() string {
	res := make(string{}, len(s.Lines))
	for i, line := range s.Lines {
		operation := "?"
		switch s.Type {
		case "ADDED":
			operation += "+"
		case "REMOVED":
			operation += "-"
		case "CONTEXT":
			operation += " "
		}

		res[i] = operation + line.String()
	}

	return strings.Join(res, "\n")
}

func (l Line) String() string {
	res := ""
	res += line.Line

	comments := [len(line.Comments)]string{}
	for i, comment := range line.Comments {
		comments[i] = comment.String()
	}
	res += strings.Join(comments, "\n")

	return res
}

func (c Comment) String() string {
	return c.Text
}
