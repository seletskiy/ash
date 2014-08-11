package main

import (
	"bytes"
	"text/template"
)

type Hunk struct {
	SourceLine      int64
	SourceSpan      int64
	DestinationLine int64
	DestinationSpan int64
	Truncated       bool
	Segments        []*Segment
}

var hunkTpl = template.Must(template.New("diff").Parse(
	"@@ -{{.SourceLine}},{{.SourceSpan}} " +
		"+{{.DestinationLine}},{{.DestinationSpan}} @@\n" +
		"{{range .Segments}}" +
		"{{.}}" +
		"{{end}}"))

func (h Hunk) String() string {
	buf := bytes.NewBuffer([]byte{})
	hunkTpl.Execute(buf, h)

	return buf.String()
}
