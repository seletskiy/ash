package main

import (
	"log"
	"os"
	"reflect"
	"testing"

	"github.com/seletskiy/godiff"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		fromFile string
		toFile   string
		expected []map[string]interface{}
	}{
		{
			"_test/without_comments.diff",
			"_test/with_one_comment.diff",
			[]map[string]interface{}{
				{
					"text": "hello",
					"anchor": map[string]interface{}{
						"line":     int64(3),
						"lineType": godiff.SegmentTypeAdded,
						"path":     "/tmp/a",
						"srcPath":  "/tmp/a",
					},
				},
			},
		},
		{
			"_test/with_one_stored_comment.diff",
			"_test/without_comments.diff",
			// TODO: fix tests so the actually check that returned value has
			// type CommentRemoved and filled properly.
			[]map[string]interface{}{nil},
		},
		{
			"_test/without_comments.diff",
			"_test/with_one_new_nested_comment.diff",
			[]map[string]interface{}{
				{
					"text": "bla",
					"parent": map[string]interface{}{
						"id": int64(1234),
					},
				},
			},
		},
		{
			"_test/with_one_nested_stored_comment.diff",
			"_test/with_one_stored_comment.diff",
			[]map[string]interface{}{nil},
		},
		{
			"_test/with_one_nested_stored_comment.diff",
			"_test/with_one_modified_nested_comment.diff",
			[]map[string]interface{}{
				{
					"text":    "bla2",
					"id":      int64(1235),
					"version": 1,
				},
			},
		},
		{
			"_test/without_comments.diff",
			"_test/with_one_new_top_level_comment.diff",
			[]map[string]interface{}{
				{
					"text": "hello there",
				},
			},
		},
	}

	for _, test := range tests {
		actual := compareTwoReviews(test.fromFile, test.toFile)

		if len(test.expected) != len(actual) {
			t.Fatalf("unexpected length of changeset: %d instead of %d",
				len(actual), len(test.expected))
			t.FailNow()
		}

		for i, c := range test.expected {
			if !reflect.DeepEqual(c, actual[i].GetPayload()) {
				t.Fatalf("two changes are not equal\n%#v\n%#v",
					c, actual[i].GetPayload())
			}
		}
	}
}

func compareTwoReviews(origFile, compareFile string) []ReviewChange {
	a, err := parseReviewFile(origFile)
	if err != nil {
		log.Fatal(err)
	}
	b, err := parseReviewFile(compareFile)
	if err != nil {
		log.Fatal(err)
	}

	return a.Compare(b)
}

func parseReviewFile(path string) (*Review, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	defer file.Close()

	return ReadReview(file)
}
