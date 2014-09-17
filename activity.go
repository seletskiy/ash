package main

import (
	"encoding/json"
	"text/template"

	"github.com/seletskiy/godiff"
	"github.com/seletskiy/tplutil"
)

var rescopedTpl = template.Must(
	template.New(`rescoped`).Funcs(tplutil.Last).Parse(tplutil.Strip(`
{{if .Added.Changesets}}
	New commits added:
	{{"\n"}}
	{{range $i, $_ := .Added.Changesets}}
		{{.DisplayId}} | {{.Author.DisplayName}} | {{.Message}}
		{{if not (last $i $.Added.Changesets)}}
		{{"\n"}}
		{{end}}
	{{end}}
{{end}}
{{if .Removed.Changesets}}
	{{"\n"}}
	Commits removed:
	{{"\n"}}
	{{range $i, $_ := .Removed.Changesets}}
		{{.DisplayId}} | {{.Author.DisplayName}} | {{.Message}}
		{{if not (last $i $.Removed.Changesets)}}
		{{"\n"}}
		{{end}}
	{{end}}
{{end}}
`)))

var openedTpl = template.Must(
	template.New(`rescoped`).Parse(tplutil.Strip(`
Pull request opened by: {{.User.DisplayName}} <{{.User.EmailAddress}}>
`)))

var commentOnFileTpl = template.Must(
	template.New(`filecomment`).Parse(tplutil.Strip(`
{{.Comment.Author.DisplayName}} commented on file {{.CommentAnchor.Path}}:
`)))

type ReviewActivity struct {
	godiff.Changeset
}

type reviewCommented struct {
	diff *godiff.Diff
}

type reviewRescoped struct {
	diff *godiff.Diff
}

type reviewOpened struct {
	diff *godiff.Diff
}

type rescopedChangeset struct {
	CreatedDate int64
	Id          string
	DisplayId   string
	Author      struct {
		Id           int
		Name         string
		EmailAddress string
		DisplayName  string
	}
	Message string
}

type reviewMerged struct {
	diff *godiff.Diff
}

func (activity *ReviewActivity) UnmarshalJSON(data []byte) error {
	response := struct {
		Size       int
		Limit      int
		IsLastPage bool

		Values []json.RawMessage `json:"values"`
	}{}

	logger.Debug("unmarshalling activity response")

	err := json.Unmarshal(data, &response)

	if err != nil {
		return err
	}

	for _, rawActivity := range response.Values {
		head := struct{ Action string }{}
		err := json.Unmarshal(rawActivity, &head)
		if err != nil {
			return err
		}

		var diff *godiff.Diff

		switch head.Action {
		case "COMMENTED":
			value := reviewCommented{}
			err = json.Unmarshal(rawActivity, &value)
			diff = value.diff
		case "RESCOPED":
			value := reviewRescoped{}
			err = json.Unmarshal(rawActivity, &value)
			diff = value.diff
		case "MERGED":
			value := reviewMerged{}
			err = json.Unmarshal(rawActivity, &value)
			diff = value.diff
		case "OPENED":
			value := reviewOpened{}
			err = json.Unmarshal(rawActivity, &value)
			diff = value.diff
		default:
			logger.Warning("unknown activity action: %s", head.Action)
		}

		if diff != nil {
			if err != nil {
				return err
			}

			activity.Changeset.Diffs = append(activity.Changeset.Diffs, diff)
		}
	}

	return nil
}

func (rc *reviewCommented) UnmarshalJSON(data []byte) error {
	value := struct {
		Comment       *godiff.Comment
		CommentAnchor *godiff.CommentAnchor
		Diff          *godiff.Diff
	}{}

	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	// in case of comment to overall review or file, not to line
	if value.Diff == nil {
		rc.diff = &godiff.Diff{
			FileComments: godiff.CommentsTree{value.Comment},
		}

		// in case of comment to file
		if anchor := value.CommentAnchor; anchor != nil {
			if anchor.Path != "" && anchor.LineType == "" {
				rc.diff.Source.ToString = anchor.SrcPath
				rc.diff.Destination.ToString = anchor.Path
				rc.diff.Note, _ = tplutil.ExecuteToString(commentOnFileTpl,
					value)
			}
		}

		return nil
	}

	value.Diff.Source = value.Diff.Destination

	// in case of line comment or reply
	value.Diff.ForEachLine(
		func(
			diff *godiff.Diff,
			_ *godiff.Hunk,
			s *godiff.Segment,
			l *godiff.Line,
		) {
			if value.CommentAnchor.LineType != s.Type {
				return
			}

			if s.GetLineNum(l) != value.CommentAnchor.Line {
				return
			}

			l.Comments = append(l.Comments, value.Comment)
			value.Diff.LineComments = append(value.Diff.LineComments,
				value.Comment)
		})

	rc.diff = value.Diff

	return nil
}

func (rr *reviewOpened) UnmarshalJSON(data []byte) error {
	value := struct {
		CreatedDate int64
		User        struct {
			EmailAddress string
			DisplayName  string
		}
	}{}

	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	result, err := tplutil.ExecuteToString(openedTpl, value)
	rr.diff = &godiff.Diff{
		Note: result,
	}

	return err
}

func (rr *reviewRescoped) UnmarshalJSON(data []byte) error {
	value := struct {
		CreatedDate      int64
		FromHash         string
		PreviousFromHash string
		PreviousToHash   string
		ToHash           string
		Added            struct {
			Changesets []rescopedChangeset
		}
		Removed struct {
			Changesets []rescopedChangeset
		}
	}{}

	err := json.Unmarshal(data, &value)
	if err != nil {
		return err
	}

	result, err := tplutil.ExecuteToString(rescopedTpl, value)
	rr.diff = &godiff.Diff{
		Note: result,
	}

	return err
}
