package main

import "regexp"

type Line struct {
	Destination    int64
	Source         int64
	Line           string
	Truncated      bool
	ConflictMarker string
	CommentIds     []int64
	Comments       []*Comment
}

var danglingSpacesRe = regexp.MustCompile("(?m) +$")
var begOfLineRe = regexp.MustCompile("(?m)\n")

func (l Line) String() string {
	res := ""

	if len(l.Comments) > 0 {
		res = "\n---"
	}

	for _, comment := range l.Comments {
		res += comment.String()
	}

	if res != "" {
		return l.Line + danglingSpacesRe.ReplaceAllString(
			begOfLineRe.ReplaceAllString(res, "\n# "), "")
	} else {
		return l.Line
	}
}
