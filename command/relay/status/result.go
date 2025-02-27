package status

import (
	"bytes"
	"fmt"

	"github.com/EdgeMatrixChain/edge-matrix-computing/command/helper"
)

type PeersStatusResult struct {
	ID          string   `json:"id"`
	Addresses   []string `json:"addresses"`
	Reservation string   `json:"reservation"`
}

func (r *PeersStatusResult) GetOutput() string {
	var buffer bytes.Buffer

	buffer.WriteString("\n[CONNECTION STATUS]\n")
	buffer.WriteString(helper.FormatKV([]string{
		fmt.Sprintf("ID|%s", r.ID),
		fmt.Sprintf("Addresses|%s", r.Addresses),
		fmt.Sprintf("Reservation|%s", r.Reservation),
	}))
	buffer.WriteString("\n")

	return buffer.String()
}
