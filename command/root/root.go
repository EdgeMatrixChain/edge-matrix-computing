package root

import (
	"fmt"
	"github.com/emc-protocol/edge-matrix-computing/command/genesis"
	"github.com/emc-protocol/edge-matrix-computing/command/helper"
	"github.com/emc-protocol/edge-matrix-computing/command/miner"
	"github.com/emc-protocol/edge-matrix-computing/command/peers"
	"github.com/emc-protocol/edge-matrix-computing/command/relay"
	"github.com/emc-protocol/edge-matrix-computing/command/secrets"
	"github.com/emc-protocol/edge-matrix-computing/command/server"
	"github.com/emc-protocol/edge-matrix-computing/command/version"
	"os"

	"github.com/spf13/cobra"
)

type RootCommand struct {
	baseCmd *cobra.Command
}

func NewRootCommand() *RootCommand {
	rootCommand := &RootCommand{
		baseCmd: &cobra.Command{
			Short: "Edge Matrix is a framework for building edge computing networks",
		},
	}

	helper.RegisterJSONOutputFlag(rootCommand.baseCmd)
	rootCommand.baseCmd.CompletionOptions.HiddenDefaultCmd = true
	rootCommand.registerSubCommands()

	return rootCommand
}

func (rc *RootCommand) registerSubCommands() {
	rc.baseCmd.AddCommand(
		genesis.GetCommand(),
		version.GetCommand(),
		secrets.GetCommand(),
		server.GetCommand(),
		peers.GetCommand(),
		relay.GetCommand(),
		miner.GetCommand(),
	)
}

func (rc *RootCommand) Execute() {
	if err := rc.baseCmd.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)

		os.Exit(1)
	}
}
