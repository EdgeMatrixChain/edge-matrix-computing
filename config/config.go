package config

import (
	"encoding/json"
	"os"
)

// Config is the blockchain config configuration
type GenesisConfig struct {
	Name        string   `json:"name"`
	Bootnodes   []string `json:"bootnodes,omitempty"`
	Relaynodes  []string `json:"relaynodes,omitempty"`
	TeleVersion string   `json:"tele_version,omitempty""`
}

func Import(chain string) (*GenesisConfig, error) {
	return ImportFromFile(chain)
}

// ImportFromFile imports a config from a filepath
func ImportFromFile(filename string) (*GenesisConfig, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return importConfig(data)
}

func importConfig(content []byte) (*GenesisConfig, error) {
	var genesisConfig *GenesisConfig

	if err := json.Unmarshal(content, &genesisConfig); err != nil {
		return nil, err
	}
	
	return genesisConfig, nil
}
