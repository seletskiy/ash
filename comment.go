package main

import (
	"bytes"
	"regexp"
	"text/template"
	"time"
)

type UnixTimestamp int

func (u UnixTimestamp) String() string {
	return time.Unix(int64(u/1000), 0).Format(time.ANSIC)
}

type Comment struct {
	Id          int
	Version     int
	Text        string
	CreatedDate int
	UpdatedDate UnixTimestamp
	Comments    []*Comment
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

const REPLY_INDENT = "    "

var endOfLineRe = regexp.MustCompile("(?m)^")
var commentTpl = template.Must(template.New("comment").Parse(
	"\n" +
		"[{{.Id}}] | {{.Author.DisplayName}} | {{.UpdatedDate}}\n" +
		"\n" +
		"{{.Text}}\n" +
		"\n" +
		"---"))

func (c Comment) String() string {
	buf := bytes.NewBuffer([]byte{})
	commentTpl.Execute(buf, c)

	for _, reply := range c.Comments {
		buf.WriteString("\n")
		buf.WriteString(endOfLineRe.ReplaceAllLiteralString(
			reply.String(), REPLY_INDENT))
	}

	return buf.String()
}
