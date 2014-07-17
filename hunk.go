package main

import (
	"bytes"
	"text/template"
)

type Hunk struct {
	SourceLine      int
	SourceSpan      int
	DestinationLine int
	DestinationSpan int
	Truncated       bool
	Segments        []*Segment
}

var hunkTpl = template.Must(template.New("diff").Parse(
	"{{range .Segments}}" +
		"{{.}}" +
		"{{end}}"))

func (h Hunk) String() string {
	buf := bytes.NewBuffer([]byte{})
	hunkTpl.Execute(buf, h)

	return buf.String()
}
