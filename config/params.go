package config

import (
	"github.com/emc-protocol/edge-matrix-core/core/types"
)

// Params are all the set of params for the config
type Params struct {
	NetworkID  int64                  `json:"networkID"`
	Engine     map[string]interface{} `json:"engine"`
	Whitelists *Whitelists            `json:"whitelists,omitempty"`
}

func (p *Params) GetEngine() string {
	// We know there is already one
	for k := range p.Engine {
		return k
	}

	return ""
}

// Whitelists specifies supported whitelists
type Whitelists struct {
	Deployment []types.Address `json:"deployment,omitempty"`
}
