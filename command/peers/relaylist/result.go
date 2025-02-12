package relaylist

import (
	"bytes"
	"fmt"

	"github.com/emc-protocol/edge-matrix-computing/command/helper"
	"github.com/emc-protocol/edge-matrix-computing/server/proto"
)

type PeersListResult struct {
	Peers []string `json:"peers"`
}

func newPeersListResult(peers []*proto.Peer) *PeersListResult {
	resultPeers := make([]string, len(peers))
	for i, p := range peers {
		resultPeers[i] = p.Id
	}

	return &PeersListResult{
		Peers: resultPeers,
	}
}

func (r *PeersListResult) GetOutput() string {
	var buffer bytes.Buffer

	buffer.WriteString("\n[RELAY NODE LIST]\n")

	if len(r.Peers) == 0 {
		buffer.WriteString("No peers found")
	} else {
		buffer.WriteString(fmt.Sprintf("Number of peers: %d\n\n", len(r.Peers)-1))

		rows := make([]string, len(r.Peers))
		for i, p := range r.Peers {
			rows[i] = fmt.Sprintf("[%d]|%s", i, p)
		}
		buffer.WriteString(helper.FormatKV(rows))
	}

	buffer.WriteString("\n")

	return buffer.String()
}
