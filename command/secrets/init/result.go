package init

import (
	"bytes"
	"fmt"

	"github.com/EdgeMatrixChain/edge-matrix-computing/command/helper"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/types"
)

type SecretsInitResult struct {
	Address  types.Address `json:"address"`
	NodeID   string        `json:"node_id"`
	Insecure bool          `json:"insecure"`
}

func (r *SecretsInitResult) GetOutput() string {
	var buffer bytes.Buffer

	vals := make([]string, 0, 3)

	vals = append(
		vals,
		fmt.Sprintf("Public key (address)|%s", r.Address.String()),
	)

	vals = append(vals, fmt.Sprintf("Node ID|%s", r.NodeID))

	buffer.WriteString("\n[SECRETS INIT]\n")
	buffer.WriteString(helper.FormatKV(vals))
	buffer.WriteString("\n")

	return buffer.String()
}
