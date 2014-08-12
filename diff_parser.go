package main

import (
	"bufio"
	"errors"
	"io"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	stateStartOfFile   = "stateStartOfFile"
	stateDiffHeader    = "stateDiffHeader"
	stateHunkHeader    = "stateHunkHeader"
	stateHunkBody      = "stateHunkBody"
	stateComment       = "stateComment"
	stateCommentDelim  = "stateCommentDelim"
	stateCommentHeader = "stateCommentHeader"
)

var (
	reFromFile      = regexp.MustCompile(`^--- ([^ ]+)\t(.*)`)
	reToFile        = regexp.MustCompile(`^\+\+\+ ([^ ]+)\t(.*)`)
	reHunk          = regexp.MustCompile(`^@@ -(\d+),(\d+) \+(\d+),(\d+) @@`)
	reCommentDelim  = regexp.MustCompile(`^#\s+---`)
	reCommentHeader = regexp.MustCompile(`^#\s+\[(\d+)\]\s+\|([^|]+)\|(.*)`)
	reCommentText   = regexp.MustCompile(`^#\s*(.*)`)
	reIndent        = regexp.MustCompile(`^#(\s+)`)
)

type parser struct {
	state       string
	diffs       Diffs
	diff        *Diff
	hunk        *Hunk
	segment     *Segment
	comment     *Comment
	line        *Line
	segmentType string
}

func ParseDiff(r io.Reader) (Diffs, error) {
	buffer := bufio.NewReader(r)

	p := parser{}
	p.state = stateStartOfFile

	for {
		line, err := buffer.ReadString('\n')
		if err != nil {
			break
		}

		//fmt.Printf("[%20s] -> ", p.state)
		p.switchState(line)
		p.createNodes(line)
		p.locateNodes(line)
		p.parseLine(line)
		//fmt.Printf("[%20s] |%s\n", p.state, strings.TrimRight(line, "\n"))
	}

	for _, diffs := range p.diffs.Diffs {
		for _, comment := range diffs.LineComments {
			comment.Text = strings.TrimSpace(comment.Text)
		}

	}

	return p.diffs, nil
}

func (p *parser) switchState(line string) error {
	inComment := false
	switch p.state {
	case stateStartOfFile:
		switch line[0] {
		case '-':
			p.state = stateDiffHeader
		}
	case stateDiffHeader:
		switch line[0] {
		case '@':
			p.state = stateHunkHeader
		}
	case stateHunkHeader:
		p.state = stateHunkBody
		fallthrough
	case stateHunkBody, stateComment, stateCommentDelim, stateCommentHeader:
		switch line[0] {
		case ' ':
			p.state = stateHunkBody
			p.segmentType = SegmentTypeContext
		case '-':
			p.state = stateHunkBody
			p.segmentType = SegmentTypeRemoved
		case '+':
			p.state = stateHunkBody
			p.segmentType = SegmentTypeAdded
		case '@':
			p.state = stateHunkHeader
		case '#':
			inComment = true
			switch {
			case reCommentDelim.MatchString(line):
				p.state = stateCommentDelim
			case reCommentHeader.MatchString(line):
				p.state = stateCommentHeader
			case reCommentText.MatchString(line):
				p.state = stateComment
			}
		}
	}

	if !inComment {
		p.comment = nil
	}

	return nil
}

func (p *parser) createNodes(line string) error {
	switch p.state {
	case stateDiffHeader, stateHunkHeader:
		switch line[0] {
		case '-':
			p.diffs = Diffs{}
			p.diff = &Diff{}
			p.diffs.Diffs = append(p.diffs.Diffs, p.diff)
		case '@':
			p.hunk = &Hunk{}
			p.segment = &Segment{}
		}
	case stateCommentDelim:
		p.comment = &Comment{}
	case stateComment:
		switch {
		case reCommentDelim.MatchString(line):
			// noop
		//case reCommentHeader.MatchString(line):
		//    // noop
		case reCommentText.MatchString(line):
			if p.comment == nil {
				p.comment = &Comment{}
			}
		}
	case stateHunkBody:
		if p.segment.Type != p.segmentType {
			p.segment = &Segment{Type: p.segmentType}
			p.hunk.Segments = append(p.hunk.Segments, p.segment)
		}

		p.line = &Line{}
		p.segment.Lines = append(p.segment.Lines, p.line)
	}

	return nil
}

func (p *parser) locateNodes(line string) error {
	switch p.state {
	case stateComment:
		p.locateComment(line)
	case stateHunkBody:
		p.locateLine(line)
	}

	return nil
}

func (p *parser) locateComment(line string) error {
	if p.comment.Parented || strings.TrimSpace(line) == "#" {
		return nil
	}

	p.diff.LineComments = append(p.diff.LineComments, p.comment)
	switch p.segment.Type {
	case SegmentTypeContext:
		p.comment.Anchor.LineType = SegmentTypeContext
		p.comment.Anchor.Line = p.line.Source
	case SegmentTypeRemoved:
		p.comment.Anchor.LineType = SegmentTypeRemoved
		p.comment.Anchor.Line = p.line.Source
	case SegmentTypeAdded:
		p.comment.Anchor.LineType = SegmentTypeAdded
		p.comment.Anchor.Line = p.line.Destination
	}
	p.comment.Parented = true
	parent := p.findParentComment(p.comment)
	if parent != nil {
		parent.Comments = append(parent.Comments, p.comment)
	} else {
		p.line.Comments = append(p.line.Comments, p.comment)
	}

	return nil
}

func (p *parser) locateLine(line string) error {
	sourceOffset := p.hunk.SourceLine - 1
	destinationOffset := p.hunk.DestinationLine - 1
	if len(p.hunk.Segments) > 1 {
		prevSegment := p.hunk.Segments[len(p.hunk.Segments)-2]
		lastLine := prevSegment.Lines[len(prevSegment.Lines)-1]
		sourceOffset = lastLine.Source
		destinationOffset = lastLine.Destination
	}
	hunkLength := int64(len(p.segment.Lines))
	switch p.segment.Type {
	case SegmentTypeContext:
		p.line.Source = sourceOffset + hunkLength
		p.line.Destination = destinationOffset + hunkLength
	case SegmentTypeAdded:
		p.line.Source = sourceOffset
		p.line.Destination = destinationOffset + hunkLength
	case SegmentTypeRemoved:
		p.line.Source = sourceOffset + hunkLength
		p.line.Destination = destinationOffset
	}

	return nil
}

func (p *parser) parseLine(line string) error {
	switch p.state {
	case stateDiffHeader:
		p.parseDiffHeader(line)
	case stateHunkHeader:
		p.parseHunkHeader(line)
	case stateHunkBody:
		p.parseHunkBody(line)
	case stateComment:
		p.parseComment(line)
	case stateCommentHeader:
		p.parseCommentHeader(line)
	}

	return nil
}

func (p *parser) parseDiffHeader(line string) error {
	switch {
	case reFromFile.MatchString(line):
		matches := reFromFile.FindStringSubmatch(line)
		p.diffs.Path = matches[1]
		p.diffs.FromHash = matches[2]
	case reToFile.MatchString(line):
		matches := reToFile.FindStringSubmatch(line)
		p.diffs.ToHash = matches[2]
	default:
		return errors.New("expected diff header, but not found")
	}
	return nil
}

func (p *parser) parseHunkHeader(line string) error {
	matches := reHunk.FindStringSubmatch(line)
	p.hunk.SourceLine, _ = strconv.ParseInt(matches[1], 10, 16)
	p.hunk.SourceSpan, _ = strconv.ParseInt(matches[2], 10, 16)
	p.hunk.DestinationLine, _ = strconv.ParseInt(matches[3], 10, 16)
	p.hunk.DestinationSpan, _ = strconv.ParseInt(matches[4], 10, 16)
	p.diff.Hunks = append(p.diff.Hunks, p.hunk)

	return nil
}

func (p *parser) parseHunkBody(line string) error {
	p.line.Line = line[1 : len(line)-1]
	return nil
}

func (p *parser) parseCommentHeader(line string) error {
	matches := reCommentHeader.FindStringSubmatch(line)
	p.comment.Author.DisplayName = strings.TrimSpace(matches[2])
	p.comment.Id, _ = strconv.ParseInt(matches[1], 10, 16)
	updatedDate, _ := time.ParseInLocation(time.ANSIC,
		strings.TrimSpace(matches[3]),
		time.Local)
	p.comment.UpdatedDate = UnixTimestamp(updatedDate.Unix() * 1000)
	p.comment.Indent = getIndentSize(line)

	return nil
}

func (p *parser) parseComment(line string) error {
	matches := reCommentText.FindStringSubmatch(line)
	p.comment.Text += "\n" + strings.Trim(matches[1], " \t")

	return nil
}

func (p *parser) findParentComment(comment *Comment) *Comment {
	for i := len(p.diff.LineComments) - 1; i >= 0; i-- {
		c := p.diff.LineComments[i]
		if comment.Indent > c.Indent {
			return c
		}
	}

	return nil
}

func getIndentSize(line string) int {
	matches := reIndent.FindStringSubmatch(line)
	if len(matches) == 0 {
		return 0
	}

	return len(matches[1])
}
