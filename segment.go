package main

import "strings"

type Segment struct {
	Type      string
	Truncated bool
	Lines     []*Line
}

func (s Segment) String() string {
	res := make([]string, len(s.Lines))
	for i, line := range s.Lines {
		operation := "?"
		switch s.Type {
		case "ADDED":
			operation = "+"
		case "REMOVED":
			operation = "-"
		case "CONTEXT":
			operation = " "
		}

		res[i] = operation + line.String()
	}

	return strings.Join(res, "\n")
}
