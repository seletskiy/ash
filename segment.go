package main

const (
	SegmentTypeContext = "CONTEXT"
	SegmentTypeRemoved = "REMOVED"
	SegmentTypeAdded   = "ADDED"
)

type Segment struct {
	Type      string
	Truncated bool
	Lines     []*Line
}

func (s Segment) String() string {
	result := ""
	for _, line := range s.Lines {
		operation := "?"
		switch s.Type {
		case SegmentTypeAdded:
			operation = "+"
		case SegmentTypeRemoved:
			operation = "-"
		case SegmentTypeContext:
			operation = " "
		}

		result += operation + line.String() + "\n"
	}

	return result
}
