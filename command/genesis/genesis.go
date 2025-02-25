package genesis

import (
	"fmt"

	"github.com/emc-protocol/edge-matrix-computing/command"
	"github.com/emc-protocol/edge-matrix-computing/command/helper"
	"github.com/spf13/cobra"
)

func GetCommand() *cobra.Command {
	genesisCmd := &cobra.Command{
		Use:     "genesis",
		Short:   "Generates the genesis configuration file with the passed in parameters",
		PreRunE: runPreRun,
		Run:     runCommand,
	}

	helper.RegisterGRPCAddressFlag(genesisCmd)

	setFlags(genesisCmd)

	return genesisCmd
}

func setFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(
		&params.genesisPath,
		dirFlag,
		fmt.Sprintf("./%s", command.DefaultGenesisFileName),
		"the directory for the Edge Matrix genesis data",
	)

	cmd.Flags().StringVar(
		&params.name,
		nameFlag,
		"EdgeMatrixComputing",
		"the name of p2p network",
	)

	cmd.Flags().Int64Var(
		&params.networkId,
		networkIdFlag,
		0,
		"the id of p2p network",
	)

	cmd.Flags().StringArrayVar(
		&params.bootNodes,
		command.BootnodeFlag,
		[]string{},
		"multiAddr URL for p2p discovery bootstrap. This flag can be used multiple times",
	)

	cmd.Flags().StringArrayVar(
		&params.relayNodes,
		command.RelaynodeFlag,
		[]string{},
		"multiAddr URL for relay discovery bootstrap. This flag can be used multiple times",
	)

}

func runPreRun(cmd *cobra.Command, _ []string) error {
	if err := params.validateFlags(); err != nil {
		return err
	}

	helper.SetRequiredFlags(cmd, params.getRequiredFlags())

	return params.initRawParams()
}

func runCommand(cmd *cobra.Command, _ []string) {
	outputter := command.InitializeOutputter(cmd)
	defer outputter.WriteOutput()

	var err error

	err = params.generateGenesis()

	if err != nil {
		outputter.SetError(err)

		return
	}

	outputter.SetCommandResult(params.getResult())
}
