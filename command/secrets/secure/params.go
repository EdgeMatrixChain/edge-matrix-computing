package secure

import (
	"errors"

	"github.com/emc-protocol/edge-matrix-computing/command"
	"github.com/emc-protocol/edge-matrix-core/core/secrets"
	"github.com/emc-protocol/edge-matrix-core/core/secrets/helper"
)

const (
	dataDirFlag    = "data-dir"
	configFlag     = "config"
	ecdsaFlag      = "ecdsa"
	blsFlag        = "bls"
	networkFlag    = "network"
	icpFlag        = "icp"
	localStoreFlag = "local-storage"
	numFlag        = "num"
)

var (
	errInvalidConfig                  = errors.New("invalid secrets configuration")
	errInvalidParams                  = errors.New("no config file or data directory passed in")
	errUnsupportedType                = errors.New("unsupported secrets manager")
	errSecureLocalStoreNotImplemented = errors.New(
		"use a secrets backend, or supply an --insecure flag " +
			"to store the private keys locally on the filesystem, " +
			"avoid doing so in production")
)

type initParams struct {
	dataDir              string
	configPath           string
	generatesECDSA       bool
	generatesBLS         bool
	generatesNetwork     bool
	generatesICPIdentity bool

	ensecureLocalStore bool

	secretsManager secrets.SecretsManager
	secretsConfig  *secrets.SecretsManagerConfig
}

func (ip *initParams) validateFlags() error {
	if ip.dataDir == "" && ip.configPath == "" {
		return errInvalidParams
	}

	return nil
}

func (ip *initParams) encryptSecrets(secretsPass string) error {
	if err := ip.initSecretsManager(); err != nil {
		return err
	}

	if err := ip.encryptValidatorKey(secretsPass); err != nil {
		return err
	}

	return ip.encryptNetworkingKey(secretsPass)
}

func (ip *initParams) initSecretsManager() error {
	return ip.encryptLocalSecretsManager()
}

func (ip *initParams) hasConfigPath() bool {
	return ip.configPath != ""
}

func (ip *initParams) parseConfig() error {
	secretsConfig, readErr := secrets.ReadConfig(ip.configPath)
	if readErr != nil {
		return errInvalidConfig
	}

	if !secrets.SupportedServiceManager(secretsConfig.Type) {
		return errUnsupportedType
	}

	ip.secretsConfig = secretsConfig

	return nil
}

func (ip *initParams) encryptLocalSecretsManager() error {
	if !ip.ensecureLocalStore {
		//Storing secrets on a local file system should only be allowed with --insecure flag,
		//to raise awareness that it should be only used in development/testing environments.
		//Production setups should use one of the supported secrets managers
		return errSecureLocalStoreNotImplemented
	}

	// setup local secrets manager
	local, err := helper.SetupLocalSecretsManager(ip.dataDir)
	if err != nil {
		return err
	}

	ip.secretsManager = local

	return nil
}

func (ip *initParams) encryptValidatorKey(secretsPass string) error {
	var err error

	if ip.generatesECDSA {
		if err = helper.EncryptECDSAValidatorKey(ip.secretsManager, secretsPass); err != nil {
			return err
		}
	}

	if ip.generatesBLS {
		if err = helper.EncryptBLSValidatorKey(ip.secretsManager, secretsPass); err != nil {
			return err
		}
	}

	if ip.generatesICPIdentity {
		if err = helper.EncryptICPIdentityKey(ip.secretsManager, secretsPass); err != nil {
			return err
		}
	}

	return nil
}

func (ip *initParams) encryptNetworkingKey(secretsPass string) error {
	if ip.generatesNetwork {
		if err := helper.EncryptNetworkingPrivateKey(ip.secretsManager, secretsPass); err != nil {
			return err
		}
	}

	return nil
}

// getResult gets keys from secret manager and return result to display
func (ip *initParams) getResult() (command.CommandResult, error) {
	var (
		res = &SecretsInitResult{}
		err error
	)

	if res.Address, err = helper.LoadValidatorAddress(ip.secretsManager); err != nil {
		return nil, err
	}

	if res.BLSPubkey, err = helper.LoadBLSPublicKey(ip.secretsManager); err != nil {
		return nil, err
	}

	if res.NodeID, err = helper.LoadNodeID(ip.secretsManager); err != nil {
		return nil, err
	}

	res.Ensecure = ip.ensecureLocalStore

	return res, nil
}
