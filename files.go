package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kovetskiy/hierr"
)

type ReviewFiles []ReviewFile

type ReviewFile struct {
	Name       string
	Parent     string
	SrcPath    string
	DstPath    string
	Type       string
	ChangeType string
	SrcExec    bool
	DstExec    bool
	Unchanged  int
}

func (rf *ReviewFiles) UnmarshalJSON(data []byte) error {
	response := struct {
		Values []struct {
			Path struct {
				Parent   string
				Name     string
				ToString string
			}
			Executable       bool
			PercentUnchanged int
			Type             string
			NodeType         string
			SrcPath          struct {
				ToString string
			}
			SrcExecutable bool
		}
	}{}

	err := json.Unmarshal(data, &response)
	if err != nil {
		return err
	}

	for _, change := range response.Values {
		*rf = append(*rf, ReviewFile{
			Name:       change.Path.Name,
			Parent:     change.Path.Parent,
			SrcPath:    change.SrcPath.ToString,
			DstPath:    change.Path.ToString,
			Type:       change.NodeType,
			ChangeType: change.Type,
			SrcExec:    change.SrcExecutable,
			DstExec:    change.Executable,
		})
	}

	return nil
}

type ErrorPointer struct {
	Pointer interface{}
}

func (err ErrorPointer) String() string {
	return (*err.Pointer.(*error)).Error()
}

func (rf ReviewFiles) String() string {
	rootPrefix := ""
	entryRoot := fmt.Errorf(".")
	entries := map[string]*error{
		rootPrefix: &entryRoot,
	}

	for _, file := range rf {

		dirname := file.Parent
		for {
			dirname = filepath.Dir(dirname)

			if _, ok := entries[dirname]; ok {
				break
			}

			if dirname == "." {
				dirname = rootPrefix
				break
			}
		}

		if _, ok := entries[file.Parent]; !ok {
			parent := fmt.Errorf(
				strings.TrimPrefix(file.Parent, dirname+"/"),
			)
			entries[file.Parent] = &parent

			newRoot := hierr.Push(
				*entries[dirname],
				ErrorPointer{&parent},
			)
			*entries[dirname] = newRoot

		}

		filename := file.Name

		switch file.ChangeType {
		case "ADD":
			filename = "A " + filename
		case "MODIFY":
			filename = "M " + filename
		case "DELETE":
			filename = "D " + filename
		case "MOVE":
			filename = "R " + filename
		}

		err := hierr.Push(
			*entries[file.Parent],
			filename,
		)

		*entries[file.Parent] = err

	}

	return entryRoot.Error()
}
