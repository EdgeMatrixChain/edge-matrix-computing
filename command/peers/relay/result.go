package relay

import (
	"bytes"
	"fmt"

	"github.com/emc-protocol/edge-matrix-computing/command/helper"
)

type RelayConnectionsCountResult struct {
	Connected      uint64 `json:"connected"`
	MaxReservation int64  `json:"maxReservation"`
}

func newRelayConnectionsCountResult(connected uint64, maxReservation int64) *RelayConnectionsCountResult {
	return &RelayConnectionsCountResult{
		Connected:      connected,
		MaxReservation: maxReservation,
	}
}

func (r *RelayConnectionsCountResult) GetOutput() string {
	var buffer bytes.Buffer

	buffer.WriteString("\n[RELAY CONNECTIONS]\n")
	buffer.WriteString(helper.FormatKV([]string{
		fmt.Sprintf("MaxReservation|%d", r.MaxReservation),
		fmt.Sprintf("Connected|%d", r.Connected),
	}))
	buffer.WriteString("\n")

	return buffer.String()
}
