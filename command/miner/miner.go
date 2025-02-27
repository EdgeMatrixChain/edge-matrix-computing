package miner

import (
	"github.com/EdgeMatrixChain/edge-matrix-computing/command/helper"
	"github.com/EdgeMatrixChain/edge-matrix-computing/command/miner/register"
	"github.com/EdgeMatrixChain/edge-matrix-computing/command/miner/status"
	"github.com/spf13/cobra"
)

func GetCommand() *cobra.Command {
	minerCmd := &cobra.Command{
		Use:   "node",
		Short: "Top level Node command for interacting with the emc. Only accepts subcommands.",
	}

	helper.RegisterGRPCAddressFlag(minerCmd)

	registerSubcommands(minerCmd)

	return minerCmd
}

func registerSubcommands(baseCmd *cobra.Command) {
	baseCmd.AddCommand(
		// miner status
		status.GetCommand(),
		// miner register
		register.GetCommand(),
	)
}
