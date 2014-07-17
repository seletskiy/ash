package main

import "regexp"

type Line struct {
	Destination    int
	Source         int
	Line           string
	Truncated      bool
	ConflictMarker string
	CommentIds     []int
	Comments       []*Comment
}

func (l Line) String() string {
	re := regexp.MustCompile("(?m)\n")

	res := ""
	if len(l.Comments) > 0 {
		res = "\n---\n"
	}

	for _, comment := range l.Comments {
		res += comment.String()
	}

	if res != "" {
		return l.Line + re.ReplaceAllLiteralString(res, "\n# ")
	} else {
		return l.Line
	}
}
