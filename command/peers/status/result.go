package status

import (
	"bytes"
	"fmt"

	"github.com/EdgeMatrixChain/edge-matrix-computing/command/helper"
)

type PeersStatusResult struct {
	ID        string   `json:"id"`
	Addresses []string `json:"addresses"`
}

func (r *PeersStatusResult) GetOutput() string {
	var buffer bytes.Buffer

	buffer.WriteString("\n[PEER STATUS]\n")
	buffer.WriteString(helper.FormatKV([]string{
		fmt.Sprintf("ID|%s", r.ID),
		fmt.Sprintf("Addresses|%s", r.Addresses),
	}))
	buffer.WriteString("\n")

	return buffer.String()
}
