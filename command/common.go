package command

import (
	"errors"
)

const (
	NoDiscoverFlag = "no-discover"
	BootnodeFlag   = "bootnode"
	RelaynodeFlag  = "relaynode"
	LogLevelFlag   = "log-level"
)

var (
	errInvalidMinNumRelays = errors.New("minimum number of relay node must be greater than 0")
)
