package command

import (
	"errors"
	"github.com/emc-protocol/edge-matrix-core/core/crypto"
	"github.com/emc-protocol/edge-matrix-core/core/secrets"
)

const (
	NoDiscoverFlag = "no-discover"
	BootnodeFlag   = "bootnode"
	LogLevelFlag   = "log-level"
)

var (
	errInvalidValidatorRange = errors.New("minimum number of validators can not be greater than the " +
		"maximum number of validators")
	errInvalidMinNumValidators = errors.New("minimum number of validators must be greater than 0")
	errInvalidMaxNumValidators = errors.New("maximum number of validators must be lower or equal " +
		"than MaxSafeJSInt (2^53 - 2)")

	ErrValidatorNumberExceedsMax = errors.New("validator number exceeds max validator number")
	ErrECDSAKeyNotFound          = errors.New("ECDSA key not found in given path")
	ErrBLSKeyNotFound            = errors.New("BLS key not found in given path")
)

func getBLSPublicKeyBytesFromSecretManager(manager secrets.SecretsManager) ([]byte, error) {
	if !manager.HasSecret(secrets.ValidatorBLSKey) {
		return nil, ErrBLSKeyNotFound
	}

	keyBytes, err := manager.GetSecret(secrets.ValidatorBLSKey)
	if err != nil {
		return nil, err
	}

	secretKey, err := crypto.BytesToBLSSecretKey(keyBytes)
	if err != nil {
		return nil, err
	}

	pubKey, err := secretKey.GetPublicKey()
	if err != nil {
		return nil, err
	}

	pubKeyBytes, err := pubKey.MarshalBinary()
	if err != nil {
		return nil, err
	}

	return pubKeyBytes, nil
}
