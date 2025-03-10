package genesis

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/EdgeMatrixChain/edge-matrix-computing/command"
	"github.com/EdgeMatrixChain/edge-matrix-computing/config"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/helper/common"
)

const (
	dirFlag       = "dir"
	nameFlag      = "name"
	networkIdFlag = "network-id"
)

var (
	params = &genesisParams{}
)

type genesisParams struct {
	genesisPath string
	name        string
	networkId   int64
	bootNodes   []string
	relayNodes  []string

	genesisConfig *config.GenesisConfig
}

// WriteGenesisConfigToDisk writes the passed in configuration to a genesis file at the specified path
func writeGenesisConfigToDisk(genesisConfig *config.GenesisConfig, genesisPath string) error {
	data, err := json.MarshalIndent(genesisConfig, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to generate genesis: %w", err)
	}

	if err := common.SaveFileSafe(genesisPath, data, 0660); err != nil {
		return fmt.Errorf("failed to write genesis: %w", err)
	}

	return nil
}

func (p *genesisParams) validateFlags() error {
	// Check if the genesis file already exists
	if generateError := verifyGenesisExistence(p.genesisPath); generateError != nil {
		return errors.New(generateError.GetMessage())
	}

	return nil
}

func (p *genesisParams) getRequiredFlags() []string {

	return []string{}
}

func (p *genesisParams) initRawParams() error {
	return nil
}

func (p *genesisParams) generateGenesis() error {
	if err := p.initGenesisConfig(); err != nil {
		return err
	}

	if err := writeGenesisConfigToDisk(
		p.genesisConfig,
		p.genesisPath,
	); err != nil {
		return err
	}

	return nil
}

func (p *genesisParams) initGenesisConfig() error {
	genesisConfig := &config.GenesisConfig{
		Name:       p.name,
		NetworkId:  p.networkId,
		Bootnodes:  p.bootNodes,
		Relaynodes: p.relayNodes,
	}

	p.genesisConfig = genesisConfig

	return nil
}

func (p *genesisParams) getResult() command.CommandResult {
	return &GenesisResult{
		Message: fmt.Sprintf("Genesis written to %s\n", p.genesisPath),
	}
}
