package status

import (
	"context"
	"github.com/emc-protocol/edge-matrix-computing/command"
	"github.com/emc-protocol/edge-matrix-computing/command/helper"
	"github.com/emc-protocol/edge-matrix-computing/server/proto"
	"github.com/spf13/cobra"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

func GetCommand() *cobra.Command {
	peersStatusCmd := &cobra.Command{
		Use:   "status",
		Short: "Returns the connection status of the relay",
		Run:   runCommand,
	}

	return peersStatusCmd
}

func runCommand(cmd *cobra.Command, _ []string) {
	outputter := command.InitializeOutputter(cmd)
	defer outputter.WriteOutput()

	peerStatus, err := getRelayStatus(helper.GetGRPCAddress(cmd))
	if err != nil {
		outputter.SetError(err)

		return
	}

	outputter.SetCommandResult(&PeersStatusResult{
		ID:          peerStatus.Id,
		Addresses:   peerStatus.Addrs,
		Reservation: peerStatus.Reservation,
	})
}

func getRelayStatus(grpcAddress string) (*proto.Peer, error) {
	client, err := helper.GetSystemClientConnection(grpcAddress)
	if err != nil {
		return nil, err
	}

	return client.RelayStatus(context.Background(), &empty.Empty{})
}
