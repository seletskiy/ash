package main

import (
	"os"

	"github.com/seletskiy/godiff"
)

type Review struct {
	changeset godiff.Changeset
}

type ReviewChange map[string]interface{}

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
			return ReviewChange{
				"text": comment.Text,
				"parent": map[string]interface{}{
					"id": parent.Id,
				},
			}
		} else {
			return ReviewChange{
				"text": comment.Text,
				"anchor": map[string]interface{}{
					"line":     comment.Anchor.Line,
					"lineType": comment.Anchor.LineType,
					"fromFile": comment.Anchor.SrcPath,
				},
			}
		}
	} else {
		for i, c := range comments {
			if c != nil && c.Id == comment.Id {
				comments[i] = nil
				if c.Text != comment.Text {
					return ReviewChange{
						"text": comment.Text,
						"id":   comment.Id,
					}
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
			changes = append(changes, ReviewChange{"id": deleted.Id})
		}
	}

	return changes
}
