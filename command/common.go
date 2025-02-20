package command

import (
	"errors"
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
)
