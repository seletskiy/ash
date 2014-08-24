package main

import (
	"log"
	"reflect"
	"testing"

	"github.com/seletskiy/godiff"
)

func TestCompare(t *testing.T) {
	tests := []struct {
		fromFile string
		toFile   string
		expected []ReviewChange
	}{
		{
			"_test/without_comments.diff",
			"_test/with_one_comment.diff",
			[]ReviewChange{
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
			[]ReviewChange{
				{
					"id":      int64(1234),
					"version": 1,
				},
			},
		},
		{
			"_test/without_comments.diff",
			"_test/with_one_new_nested_comment.diff",
			[]ReviewChange{
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
			[]ReviewChange{
				{
					"id":      int64(1235),
					"version": 1,
				},
			},
		},
		{
			"_test/with_one_nested_stored_comment.diff",
			"_test/with_one_modified_nested_comment.diff",
			[]ReviewChange{
				{
					"text":    "bla2",
					"id":      int64(1235),
					"version": 1,
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
			if !reflect.DeepEqual(c, actual[i]) {
				t.Fatalf("two changes are not equal\n%#v\n%#v", c, actual[i])
			}
		}
	}
}

func compareTwoReviews(origFile, compareFile string) []ReviewChange {
	a, err := ParseReviewFile(origFile)
	if err != nil {
		log.Fatal(err)
	}
	b, err := ParseReviewFile(compareFile)
	if err != nil {
		log.Fatal(err)
	}

	return a.Compare(b)
}
