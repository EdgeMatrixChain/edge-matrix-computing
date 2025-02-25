package relay

import (
	"github.com/emc-protocol/edge-matrix-computing/command/helper"
	"github.com/emc-protocol/edge-matrix-computing/command/relay/list"
	"github.com/emc-protocol/edge-matrix-computing/command/relay/status"
	"github.com/spf13/cobra"
)

func GetCommand() *cobra.Command {
	peersCmd := &cobra.Command{
		Use:   "relay",
		Short: "Top level command for interacting with the relay. Only accepts subcommands.",
	}

	helper.RegisterGRPCAddressFlag(peersCmd)

	registerSubcommands(peersCmd)

	return peersCmd
}

func registerSubcommands(baseCmd *cobra.Command) {
	baseCmd.AddCommand(
		// relay status
		status.GetCommand(),
		// relay list
		list.GetCommand(),
	)
}
