package main

import (
	"fmt"
	"github.com/bndr/gopencils"
	"os"
)

type DiffsReply struct {
	Diffs []struct {
		Truncated bool
		Source    struct {
			Parent string
			Name   string
		}
		Destination struct {
			Parent string
			Name   string
		}
		Hunks []struct {
			SourceLine      int
			SourceSpan      int
			DestinationLine int
			DestinationSpan int
			Truncated       string
			Segments        []struct {
				Type      string
				Truncated bool
				Lines     []struct {
					Destination    int
					Source         int
					Line           string
					Truncated      bool
					ConflictMarker string
					CommentIds     []int
				}
			}
		}
		LineComments []struct {
			Id          int
			Version     int
			Text        string
			CreatedDate int
			UpdatedDate int
			Author      struct {
				Name         string
				EmailAddress string
				Id           int
				DisplayName  string
				Active       bool
				Slug         string
				Type         string
			}
			PermittedOperations struct {
				Editable  bool
				Deletable bool
			}
		}
	}
}

func main() {
	auth := gopencils.BasicAuth{os.Args[1], os.Args[2]}

	api := gopencils.Api(fmt.Sprintf(
		"http://git.rn/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d",
		"OAPP",
		"deployer",
		2,
	), &auth)

	data := &DiffsReply{}

	r, err := api.Res("diff", data).Id("deployer.go").Get()

	fmt.Println(data)
	fmt.Println(r)
	fmt.Println(err)
}
