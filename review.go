package main

import (
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/seletskiy/godiff"
)

const usageText = "Oh, hello there!\n\n" +
	"Some points about using ash:\n" +
	"* Everything beginning with ### will be ignored.\n" +
	"* Use one # to start a comment.\n" +
	"* You can add line comments after specific lines.\n" +
	"* You can add file comments outside of the diff.\n" +
	"* You can add review comments outside of the diff (in the overview mode).\n" +
	"* If you want to delete comment, you need to remove all it's contents\n" +
	"  including header."

const vimModeline = "vim: ft=diff"

var reDanglingSpace = regexp.MustCompile(`(?m)\s*$`)

type Review struct {
	changeset  godiff.Changeset
	isOverview bool
}

type ReviewChange interface {
	GetPayload() map[string]interface{}
}

type LineCommentAdded struct {
	comment *godiff.Comment
}

type FileCommentAdded struct {
	comment *godiff.Comment
}

type ReviewCommentAdded struct {
	comment *godiff.Comment
}

type ReplyAdded struct {
	comment *godiff.Comment
	parent  *godiff.Comment
}

type CommentModified struct {
	comment *godiff.Comment
}

type CommentRemoved struct {
	comment *godiff.Comment
}

func (c LineCommentAdded) GetPayload() map[string]interface{} {
	return map[string]interface{}{
		"text": c.comment.Text,
		"anchor": map[string]interface{}{
			"line":     c.comment.Anchor.Line,
			"lineType": c.comment.Anchor.LineType,
			"path":     c.comment.Anchor.Path,
			"srcPath":  c.comment.Anchor.SrcPath,
			"commitRange": map[string]interface{}{
				"pullRequest": map[string]interface{}{
					"fromRef": map[string]interface{}{
						"latestChangeset": c.comment.Anchor.FromHash,
					},
					"toRef": map[string]interface{}{
						"latestChangeset": c.comment.Anchor.ToHash,
					},
				},
				"untilRevision": map[string]interface{}{
					"id": c.comment.Anchor.ToHash,
				},
				"sinceRevision": map[string]interface{}{
					"id": c.comment.Anchor.FromHash,
				},
			},
		},
	}
}

func (c FileCommentAdded) GetPayload() map[string]interface{} {
	return map[string]interface{}{
		"text": c.comment.Text,
		"anchor": map[string]interface{}{
			"path":    c.comment.Anchor.Path,
			"srcPath": c.comment.Anchor.SrcPath,
		},
	}
}

func (c ReviewCommentAdded) GetPayload() map[string]interface{} {
	return map[string]interface{}{
		"text": c.comment.Text,
	}
}

func (c ReplyAdded) GetPayload() map[string]interface{} {
	return map[string]interface{}{
		"text": c.comment.Text,
		"parent": map[string]interface{}{
			"id": c.parent.Id,
		},
	}
}

func (c CommentModified) GetPayload() map[string]interface{} {
	return map[string]interface{}{
		"text":    c.comment.Text,
		"id":      c.comment.Id,
		"version": c.comment.Version,
	}
}

func (c CommentRemoved) GetPayload() map[string]interface{} {
	return map[string]interface{}{
		"id": c.comment.Id,
	}
}

func ReadReview(r io.Reader) (*Review, error) {
	changeset, err := godiff.ReadChangeset(r)
	if err != nil {
		return nil, err
	}

	return &Review{
		changeset:  changeset,
		isOverview: false,
	}, nil
}

func AddUsageComment(r *Review) {
	r.changeset.Diffs = append(
		[]*godiff.Diff{
			&godiff.Diff{
				Note: usageText,
			},
		},
		r.changeset.Diffs...,
	)
}

func AddAshModeline(url string, review *Review) {
	fileTag := "overview"
	if !review.isOverview {
		fileName := review.changeset.Diffs[0].Source.ToString
		if fileName == "" {
			fileName = review.changeset.Diffs[0].Destination.ToString
		}

		fileTag = fmt.Sprintf("file=%s", fileName)
	}

	review.changeset.Diffs = append(
		review.changeset.Diffs,
		&godiff.Diff{
			Note: fmt.Sprintf("ash: review-url=%s %s", url, fileTag),
		},
	)
}

func WriteReview(review *Review, writer io.Writer) error {
	return godiff.WriteChangeset(review.changeset, writer)
}

func (current *Review) Compare(another *Review) []ReviewChange {
	existComments := make([]*godiff.Comment, 0)

	current.changeset.ForEachComment(
		func(_ *godiff.Diff, comment, _ *godiff.Comment) {
			existComments = append(existComments, comment)
		})

	changes := make([]ReviewChange, 0)

	another.changeset.ForEachComment(
		func(diff *godiff.Diff, comment, parent *godiff.Comment) {
			change := matchCommentChange(existComments, comment, parent)
			if _, ok := change.(ReviewCommentAdded); ok && !current.isOverview {
				comment.Anchor.Path = diff.Destination.ToString
				comment.Anchor.SrcPath = diff.Source.ToString
				change = FileCommentAdded{comment}
			}

			if change != nil {
				changes = append(changes, change)
			}
		})

	changes = markRemovedComments(existComments, changes)

	return changes
}

func matchCommentChange(
	comments []*godiff.Comment, comment, parent *godiff.Comment,
) ReviewChange {
	if comment.Id == 0 {
		if parent != nil {
			return ReplyAdded{comment, parent}
		} else {
			if comment.Anchor.Line == 0 {
				return ReviewCommentAdded{comment}
			} else {
				return LineCommentAdded{comment}
			}
		}
	} else {
		for i, c := range comments {
			if c == nil || c.Id != comment.Id {
				continue
			}
			comments[i] = nil
			if trimCommentSpaces(c.Text) != trimCommentSpaces(comment.Text) {
				return CommentModified{comment}
			}
		}
	}

	return nil
}

func markRemovedComments(
	comments []*godiff.Comment, changes []ReviewChange,
) []ReviewChange {
	for _, deleted := range comments {
		if deleted != nil {
			changes = append(changes, CommentRemoved{deleted})
		}
	}

	return changes
}

func trimCommentSpaces(text string) string {
	return strings.TrimSpace(
		reDanglingSpace.ReplaceAllString(
			text,
			``,
		),
	)
}
