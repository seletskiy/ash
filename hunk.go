package main

import "strings"

type Hunk struct {
	SourceLine      int
	SourceSpan      int
	DestinationLine int
	DestinationSpan int
	Truncated       bool
	Segments        []*Segment
}

func (h Hunk) String() string {
	res := make([]string, len(h.Segments))
	for i, segment := range h.Segments {
		res[i] = segment.String()
	}

	return strings.Join(res, "\n")
}
