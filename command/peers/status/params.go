package status

import (
	"context"

	"github.com/EdgeMatrixChain/edge-matrix-computing/command"
	"github.com/EdgeMatrixChain/edge-matrix-computing/command/helper"
	"github.com/EdgeMatrixChain/edge-matrix-computing/server/proto"
)

var (
	params = &statusParams{}
)

const (
	peerIDFlag = "peer-id"
)

type statusParams struct {
	peerID string

	peerStatus *proto.Peer
}

func (p *statusParams) getRequiredFlags() []string {
	return []string{
		peerIDFlag,
	}
}

func (p *statusParams) initPeerInfo(grpcAddress string) error {
	systemClient, err := helper.GetSystemClientConnection(grpcAddress)
	if err != nil {
		return err
	}

	peerStatus, err := systemClient.PeersStatus(
		context.Background(),
		&proto.PeersStatusRequest{
			Id: p.peerID,
		},
	)
	if err != nil {
		return err
	}

	p.peerStatus = peerStatus

	return nil
}

func (p *statusParams) getResult() command.CommandResult {
	return &PeersStatusResult{
		ID:        p.peerStatus.Id,
		Addresses: p.peerStatus.Addrs,
	}
}
