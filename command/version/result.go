package version

import (
	"bytes"
	"fmt"
	"github.com/emc-protocol/edge-matrix-computing/command/helper"
)

type VersionResult struct {
	Version string `json:"version"`
	Branch  string `json:"branch"`
	Build   string `json:"build"`
}

func (r *VersionResult) GetOutput() string {
	var buffer bytes.Buffer

	buffer.WriteString("\n[VERSION INFO]\n")
	buffer.WriteString(helper.FormatKV([]string{
		fmt.Sprintf("Release|%s\n", r.Version),
		fmt.Sprintf("Branch|%s\n", r.Branch),
		fmt.Sprintf("Build|%s\n", r.Build),
	}))

	return buffer.String()
}
