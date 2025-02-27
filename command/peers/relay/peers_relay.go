package relay

import (
	"context"

	"github.com/EdgeMatrixChain/edge-matrix-computing/command"
	"github.com/EdgeMatrixChain/edge-matrix-computing/command/helper"
	"github.com/EdgeMatrixChain/edge-matrix-computing/server/proto"
	"github.com/spf13/cobra"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

func GetCommand() *cobra.Command {
	peersListCmd := &cobra.Command{
		Use:   "relay",
		Short: "Returns the count of clients connected to relay server",
		Run:   runCommand,
	}

	return peersListCmd
}

func runCommand(cmd *cobra.Command, _ []string) {
	outputter := command.InitializeOutputter(cmd)
	defer outputter.WriteOutput()

	relayConnectonsCount, err := getRelayConnectionsCount(helper.GetGRPCAddress(cmd))
	if err != nil {
		outputter.SetError(err)

		return
	}

	outputter.SetCommandResult(
		newRelayConnectionsCountResult(relayConnectonsCount.Connected, relayConnectonsCount.MaxReservations),
	)
}

func getRelayConnectionsCount(grpcAddress string) (*proto.RelayConnectionsCount, error) {
	client, err := helper.GetSystemClientConnection(grpcAddress)
	if err != nil {
		return nil, err
	}

	return client.RelayConnections(context.Background(), &empty.Empty{})
}
