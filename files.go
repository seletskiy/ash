package main

import "encoding/json"

type ReviewFiles []ReviewFile

type ReviewFile struct {
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
