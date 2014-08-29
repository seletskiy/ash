package main

import (
	"os"

	"github.com/seletskiy/godiff"
)

type Review struct {
	changeset godiff.Changeset
}

type ReviewChange interface {
	GetPayload() map[string]interface{}
}

type CommentAdded struct {
	comment *godiff.Comment
}

func (c CommentAdded) GetPayload() map[string]interface{} {
	return map[string]interface{}{
		"text": c.comment.Text,
		"anchor": map[string]interface{}{
			"line":     c.comment.Anchor.Line,
			"lineType": c.comment.Anchor.LineType,
			"path":     c.comment.Anchor.Path,
			"srcPath":  c.comment.Anchor.SrcPath,
		},
	}
}

type ReplyAdded struct {
	comment *godiff.Comment
	parent  *godiff.Comment
}

func (c ReplyAdded) GetPayload() map[string]interface{} {
	return map[string]interface{}{
		"text": c.comment.Text,
		"parent": map[string]interface{}{
			"id": c.parent.Id,
		},
	}
}

type CommentModified struct {
	comment *godiff.Comment
}

func (c CommentModified) GetPayload() map[string]interface{} {
	return map[string]interface{}{
		"text":    c.comment.Text,
		"id":      c.comment.Id,
		"version": c.comment.Version,
	}
}

type CommentRemoved struct {
	comment *godiff.Comment
}

func (c CommentRemoved) GetPayload() map[string]interface{} {
	return nil
}

func ParseReviewFile(path string) (*Review, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	changeset, err := godiff.ParseDiff(file)
	if err != nil {
		return nil, err
	}

	return &Review{changeset}, nil
}

func (r *Review) String() string {
	return r.changeset.String()
}

func (current *Review) Compare(another *Review) []ReviewChange {
	existComments := make([]*godiff.Comment, 0)

	current.changeset.ForEachComment(
		func(_ *godiff.Diff, comment, _ *godiff.Comment) {
			existComments = append(existComments, comment)
		})

	changes := make([]ReviewChange, 0)

	another.changeset.ForEachComment(
		func(_ *godiff.Diff, comment, parent *godiff.Comment) {
			change := matchCommentChange(existComments, comment, parent)
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
			return CommentAdded{comment}
		}
	} else {
		for i, c := range comments {
			if c != nil && c.Id == comment.Id {
				comments[i] = nil
				if c.Text != comment.Text {
					return CommentModified{comment}
				}
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
