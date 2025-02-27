package server

import (
	"github.com/EdgeMatrixChain/edge-matrix-core/core/secrets"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/secrets/awsssm"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/secrets/gcpssm"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/secrets/hashicorpvault"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/secrets/local"
)

//type GenesisFactoryHook func(config *config.Chain, engineName string) func(*state.Transition) error

// secretsManagerBackends defines the SecretManager factories for different
// secret management solutions
var secretsManagerBackends = map[secrets.SecretsManagerType]secrets.SecretsManagerFactory{
	secrets.Local:          local.SecretsManagerFactory,
	secrets.HashicorpVault: hashicorpvault.SecretsManagerFactory,
	secrets.AWSSSM:         awsssm.SecretsManagerFactory,
	secrets.GCPSSM:         gcpssm.SecretsManagerFactory,
}
