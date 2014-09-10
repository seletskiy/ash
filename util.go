package main

import (
	"bytes"
	"reflect"
	"regexp"
	"text/template"
)

type Template struct {
	*template.Template
}

var reInsignificantWhitespace = regexp.MustCompile(`(?m)\n?^\s*`)

func loadSparseTemplate(name, text string) *Template {
	stripped := reInsignificantWhitespace.ReplaceAllString(text, ``)

	funcs := template.FuncMap{
		"last": func(x int, a interface{}) bool {
			return x == reflect.ValueOf(a).Len()-1
		},
	}

	tpl := &Template{
		template.Must(template.New("comment").Funcs(funcs).Parse(stripped)),
	}

	return tpl
}

func (t *Template) Execute(v interface{}) string {
	buf := &bytes.Buffer{}
	t.Template.Execute(buf, v)

	return buf.String()
}
