package main

import (
	"encoding/json"
	"text/template"

	"github.com/seletskiy/godiff"
	"github.com/seletskiy/tplutil"
)

var updatedHeaderTpl = template.Must(
	template.New(`updated`).Parse(tplutil.Strip(`
Update at [{{.Date}}]{{"\n"}}
==={{"\n\n"}}
`)))

var rescopedTpl = template.Must(
	template.New(`rescoped`).Funcs(tplutil.Last).Parse(tplutil.Strip(`
{{$prefix := .Prefix}}
{{range $i, $_ := .Data}}
{{$prefix}} {{.DisplayId}} | {{.Author.DisplayName}} | {{.AuthorTimestamp}}{{"\n"}}
	---{{"\n"}}
	{{.Message}}{{"\n"}}
	---
	{{if not (last $i $.Data)}}
	{{"\n\n"}}
	{{end}}
{{end}}
`)))

var openedTpl = template.Must(
	template.New(`rescoped`).Parse(tplutil.Strip(`
Pull request opened by: {{.User.DisplayName}} <{{.User.EmailAddress}}>
`)))

var approvedTpl = template.Must(
	template.New(`approved`).Parse(tplutil.Strip(`
Pull request approved by: {{.User.DisplayName}} <{{.User.EmailAddress}}>
`)))

var mergedTpl = template.Must(
	template.New(`merged`).Parse(tplutil.Strip(`
Pull request merged by: {{.User.DisplayName}} <{{.User.EmailAddress}}>
`)))

var declinedTpl = template.Must(
	template.New(`declined`).Parse(tplutil.Strip(`
Pull request declined by: {{.User.DisplayName}} <{{.User.EmailAddress}}>
`)))

var reopenedTpl = template.Must(
	template.New(`reopened`).Parse(tplutil.Strip(`
Pull request reopened by: {{.User.DisplayName}} <{{.User.EmailAddress}}>
`)))

var commentOnFileTpl = template.Must(
	template.New(`filecomment`).Parse(tplutil.Strip(`
{{.Comment.Author.DisplayName}} commented on file {{.CommentAnchor.Path}}:
`)))

type ReviewActivity struct {
	godiff.Changeset
}

type reviewAction interface {
	json.Unmarshaler
	GetDiff() *godiff.Diff
}

type reviewActionBasic struct {
	diff   *godiff.Diff
	Action string
}

type reviewActionCommented struct {
	diff *godiff.Diff
}

type reviewActionRescoped struct {
	diff *godiff.Diff
}

type rescopedChangeset struct {
	AuthorTimestamp UnixTimestamp
	Id              string
	DisplayId       string
	Author          struct {
		Id           int
		Name         string
		EmailAddress string
		DisplayName  string
	}
	Message string
}

func (activity *ReviewActivity) UnmarshalJSON(data []byte) error {
	values := []json.RawMessage{}
	err := json.Unmarshal(data, &values)
	if err != nil {
		return err
	}

	for _, rawActivity := range values {
		head := struct{ Action string }{}
		err := json.Unmarshal(rawActivity, &head)
		if err != nil {
			return err
		}

		var diff *godiff.Diff
		var value reviewAction

		switch head.Action {
		case "COMMENTED":
			value = &reviewActionCommented{}
		case "RESCOPED":
			value = &reviewActionRescoped{}
		default:
			value = &reviewActionBasic{Action: head.Action}
		}

		err = json.Unmarshal(rawActivity, value)
		if err != nil {
			return err
		}

		diff = value.GetDiff()
		if diff != nil {
			activity.Changeset.Diffs = append(activity.Changeset.Diffs, diff)
		}
	}

	return nil
}

func (rc *reviewActionCommented) UnmarshalJSON(data []byte) error {
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
		) error {
			if value.CommentAnchor.LineType != s.Type {
				return nil
			}

			if s.GetLineNum(l) != value.CommentAnchor.Line {
				return nil
			}

			l.Comments = append(l.Comments, value.Comment)
			value.Diff.LineComments = append(value.Diff.LineComments,
				value.Comment)

			return nil
		})

	rc.diff = value.Diff

	return nil
}

func (rc *reviewActionCommented) GetDiff() *godiff.Diff {
	return rc.diff
}

func (rr *reviewActionRescoped) UnmarshalJSON(data []byte) error {
	value := struct {
		CreatedDate      UnixTimestamp
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

	components := []struct {
		Data   []rescopedChangeset
		Prefix string
	}{
		{value.Added.Changesets, "+"},
		{value.Removed.Changesets, "-"},
	}

	header, err := tplutil.ExecuteToString(updatedHeaderTpl, struct {
		Date UnixTimestamp
	}{
		value.CreatedDate,
	})

	if err != nil {
		return err
	}

	rr.diff = &godiff.Diff{}
	for _, val := range components {
		if len(val.Data) > 0 {
			result, err := tplutil.ExecuteToString(rescopedTpl, val)
			if err != nil {
				return err
			}

			if rr.diff.Note != "" {
				rr.diff.Note += "\n\n"
			}

			rr.diff.Note += result
		}
	}

	rr.diff.Note = header + rr.diff.Note

	return err
}

func (rr *reviewActionRescoped) GetDiff() *godiff.Diff {
	return rr.diff
}

func (rb *reviewActionBasic) UnmarshalJSON(data []byte) error {
	var tpl *template.Template
	switch rb.Action {
	case "MERGED":
		tpl = mergedTpl
	case "OPENED":
		tpl = openedTpl
	case "APPROVED":
		tpl = approvedTpl
	case "DECLINED":
		tpl = declinedTpl
	case "REOPENED":
		tpl = reopenedTpl
	default:
		logger.Warning("unknown activity action: '%s'", rb.Action)
		return nil
	}

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

	result, err := tplutil.ExecuteToString(tpl, value)

	rb.diff = &godiff.Diff{
		Note: result,
	}

	return err
}

func (rb *reviewActionBasic) GetDiff() *godiff.Diff {
	return rb.diff
}
