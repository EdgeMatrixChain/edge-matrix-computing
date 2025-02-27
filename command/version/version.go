package version

import (
	"github.com/EdgeMatrixChain/edge-matrix-computing/command"
	"github.com/EdgeMatrixChain/edge-matrix-computing/versioning"
	"github.com/spf13/cobra"
)

func GetCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Returns the current version",
		Args:  cobra.NoArgs,
		Run:   runCommand,
	}
}

func runCommand(cmd *cobra.Command, _ []string) {
	outputter := command.InitializeOutputter(cmd)
	defer outputter.WriteOutput()

	outputter.SetCommandResult(
		&VersionResult{
			Version: versioning.Version,
			Branch:  versioning.Branch,
			Build:   versioning.Build,
		},
	)
}
