package main

import (
	"bytes"
	"text/template"
)

type Diffs struct {
	Path       string
	Whitespace string
	Diffs      []*Diff
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
	Hunks        []*Hunk
	LineComments []*Comment
}

func (d Diffs) ForEachLines(callback func(*Diff, *Line)) {
	for _, diff := range d.Diffs {
		for _, hunk := range diff.Hunks {
			for _, segment := range hunk.Segments {
				for _, line := range segment.Lines {
					callback(diff, line)
				}
			}
		}
	}
}

var fileTpl = template.Must(template.New("file").Parse(
	"\n" +
		"+++ {{.Path}}\n" +
		"--- {{.Path}}\n" +
		"\n" +
		"{{range .Diffs}}" +
		"{{.}}" +
		"{{end}}"))

var diffTpl = template.Must(template.New("diff").Parse(
	"{{range .Hunks}}" +
		"{{.}}" +
		"{{end}}"))

func (d Diffs) String() string {
	buf := bytes.NewBuffer([]byte{})
	fileTpl.Execute(buf, d)

	return buf.String()
}

func (d Diff) String() string {
	buf := bytes.NewBuffer([]byte{})
	diffTpl.Execute(buf, d)

	return buf.String()
}
