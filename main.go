package main

import (
	"fmt"
	"os"

	"github.com/bndr/gopencils"
)

type DiffsReply struct {
	Whitespace string
	Diffs      []struct {
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
		//LineComments []struct {
		//    Id          int
		//    Version     int
		//    Text        string
		//    CreatedDate int
		//    UpdatedDate int
		//    Author      struct {
		//        Name         string
		//        EmailAddress string
		//        Id           int
		//        DisplayName  string
		//        Active       bool
		//        Slug         string
		//        Type         string
		//    }
		//    PermittedOperations struct {
		//        Editable  bool
		//        Deletable bool
		//    }
		//}
	}
}

func main() {
	auth := gopencils.BasicAuth{os.Args[1], os.Args[2]}

	api := gopencils.Api(fmt.Sprintf(
		"http://git.rn/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d",
		"OAPP",
		"deployer-container",
		1,
	), &auth)

	data := &DiffsReply{}

	api.Res("diff").Id("deployer-container.go", data).Get()

	//fmt.Printf("%+v", data)

	for _, h := range data.Diffs[0].Hunks {
		for _, s := range h.Segments {
			for _, l := range s.Lines {
				switch s.Type {
				case "ADDED":
					fmt.Print("+")
				case "REMOVED":
					fmt.Print("-")
				case "CONTEXT":
					fmt.Print(" ")
				}

				fmt.Println(l.Line)
			}
		}
	}
}
