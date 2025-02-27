package peers

import (
	"github.com/EdgeMatrixChain/edge-matrix-computing/command/helper"
	"github.com/EdgeMatrixChain/edge-matrix-computing/command/peers/add"
	"github.com/EdgeMatrixChain/edge-matrix-computing/command/peers/list"
	"github.com/EdgeMatrixChain/edge-matrix-computing/command/peers/relay"
	"github.com/EdgeMatrixChain/edge-matrix-computing/command/peers/status"
	"github.com/spf13/cobra"
)

func GetCommand() *cobra.Command {
	peersCmd := &cobra.Command{
		Use:   "peers",
		Short: "Top level command for interacting with the network peers. Only accepts subcommands.",
	}

	helper.RegisterGRPCAddressFlag(peersCmd)

	registerSubcommands(peersCmd)

	return peersCmd
}

func registerSubcommands(baseCmd *cobra.Command) {
	baseCmd.AddCommand(
		// status status
		status.GetCommand(),
		// peers list
		list.GetCommand(),
		// peers add
		add.GetCommand(),
		// peers relay
		relay.GetCommand(),
	)
}
