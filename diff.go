package main

import "strings"

type Diffs struct {
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

func (d Diff) String() string {
	res := make([]string, len(d.Hunks))
	for i, hunk := range d.Hunks {
		res[i] = hunk.String()
	}

	return strings.Join(res, "\n")
}
