package secrets

import (
	"github.com/emc-protocol/edge-matrix-computing/command/helper"
	"github.com/emc-protocol/edge-matrix-computing/command/secrets/generate"
	initCmd "github.com/emc-protocol/edge-matrix-computing/command/secrets/init"
	"github.com/emc-protocol/edge-matrix-computing/command/secrets/output"
	"github.com/emc-protocol/edge-matrix-computing/command/secrets/secure"
	"github.com/spf13/cobra"
)

func GetCommand() *cobra.Command {
	secretsCmd := &cobra.Command{
		Use:   "secrets",
		Short: "Top level SecretsManager command for interacting with secrets functionality. Only accepts subcommands.",
	}

	helper.RegisterGRPCAddressFlag(secretsCmd)

	registerSubcommands(secretsCmd)

	return secretsCmd
}

func registerSubcommands(baseCmd *cobra.Command) {
	baseCmd.AddCommand(
		// secrets init
		initCmd.GetCommand(),
		// secrets generate
		generate.GetCommand(),
		// secrets output public data
		output.GetCommand(),
		// secrets encrypt
		secure.GetCommand(),
	)
}
