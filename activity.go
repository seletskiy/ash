package main

import (
	"encoding/json"
	"io"
	"log"
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

type ReviewActivity interface{}
type ReviewActivities []ReviewActivity

type ReviewCommented struct {
	Comment       *godiff.Comment
	CommentAnchor *godiff.CommentAnchor
	Diff          *godiff.Diff
}

type ReviewRescoped struct {
	CreatedDate      int64
	FromHash         string
	PreviousFromHash string
	PreviousToHash   string
	ToHash           string
	Added            struct {
		Changesets []RescopedChangeset
	}
	Removed struct {
		Changesets []RescopedChangeset
	}
}

type RescopedChangeset struct {
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

type ReviewMerged struct {
	CreatedDate int64
	Author      struct {
		Id           int64
		Name         string
		EmailAddress string
		DisplayName  string
	}
}

func (activities *ReviewActivities) UnmarshalJSON(data []byte) error {
	response := struct {
		Size       int
		Limit      int
		IsLastPage bool

		Values []json.RawMessage `json:"values"`
	}{}
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

		var activity interface{} = nil
		switch head.Action {
		case "COMMENTED":
			value := ReviewCommented{}
			err = json.Unmarshal(rawActivity, &value)
			//value.BindComments()
			activity = value
		case "RESCOPED":
			value := ReviewRescoped{}
			err = json.Unmarshal(rawActivity, &value)
			activity = value
		case "MERGED":
			value := ReviewMerged{}
			err = json.Unmarshal(rawActivity, &value)
			activity = value
		default:
			log.Println("unknown activity action: %s", head.Action)
		}

		if activity != nil {
			if err != nil {
				return err
			}

			*activities = append(*activities, activity)
		}
	}

	return nil
}

func BindComment(diff *godiff.Diff, comment *godiff.Comment) *godiff.Diff {
	// in case of comment to overall review, not to line
	if diff == nil {
		return &godiff.Diff{
			diffComments: godiff.CommentsTree{comment},
		}
	}

	// in case of line comment or reply
	diff.ForEachLine(
		func(
			_ *godiff.Diff,
			_ *godiff.Hunk,
			s *godiff.Segment,
			l *godiff.Line,
		) {
			if comment.CommentAnchor.LineType != s.Type {
				return
			}

			if s.GetLineNum(l) != comment.CommentAnchor.Line {
				return
			}

			l.Comments = append(l.Comments, comment)
		})

	return diff
}

func (r ReviewCommented) ToChangeset() godiff.Changeset {
	return godiff.Changeset{
		ToHash:   r.Diff.GetHashTo(),
		FromHash: r.Diff.GetHashFrom(),
		Diffs:    []*godiff.Diff{r.Diff},
	}
}

//func (current ReviewActivities) Compare(
//    another ReviewActivities,
//) []ReviewChange {
//    existActivities := make([]ReviewCommented, 0)

//    for _, changeset := range current {
//        if commented, ok := changeset.(ReviewCommented); !ok {
//            continue
//        } else {
//            existActivities = append(existActivities, commented)
//        }
//    }

//    changes := make([]ReviewChange, 0)

//    for _, changeset := range another {
//        if commented, ok := changeset.(ReviewCommented); !ok {
//            continue
//        } else {
//            change := matchActivityChange(existActivities, commented)
//            if change != nil {
//                changes = append(changes, change)
//            }
//        }
//    }

//    //changes = markRemovedActivities(existActivities, changes)

//    return changes
//}

func ReadActivities(reader io.Reader) (ReviewActivities, error) {
	changeset, err := godiff.ReadChangeset(reader)

	if err != nil {
		return nil, err
	}

	return ReviewActivities{changeset}, nil
}

func ToChangeset() {

}

func WriteActivities(activities ReviewActivities, writer io.Writer) error {
	for index, activity := range activities {
		switch a := activity.(type) {
		case ReviewRescoped:
			data, err := tplutil.ExecuteToString(rescopedTpl, a)
			if err != nil {
				return err
			}

			writer.Write([]byte(godiff.Note(data)))
			if err != nil {
				return err
			}
		case ReviewCommented:
			err := godiff.WriteChangeset(a.ToChangeset(), writer)
			if err != nil {
				return err
			}
		}

		if index < len(activities)-1 {
			writer.Write([]byte("\n\n\n"))
		}
	}

	return nil
}
