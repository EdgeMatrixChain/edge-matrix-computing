package list

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
		Use:   "list",
		Short: "Returns the list of status nodes",
		Run:   runCommand,
	}

	return peersListCmd
}

func runCommand(cmd *cobra.Command, _ []string) {
	outputter := command.InitializeOutputter(cmd)
	defer outputter.WriteOutput()

	peersList, err := getPeersRelayList(helper.GetGRPCAddress(cmd))
	if err != nil {
		outputter.SetError(err)

		return
	}

	outputter.SetCommandResult(
		newPeersListResult(peersList.Peers),
	)
}

func getPeersRelayList(grpcAddress string) (*proto.PeersListResponse, error) {
	client, err := helper.GetSystemClientConnection(grpcAddress)
	if err != nil {
		return nil, err
	}

	return client.PeersRelayList(context.Background(), &empty.Empty{})
}
