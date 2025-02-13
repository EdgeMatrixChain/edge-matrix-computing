package server

import (
	"github.com/emc-protocol/edge-matrix-computing/config"
	"github.com/emc-protocol/edge-matrix-core/core/network"
	"net"

	"github.com/hashicorp/go-hclog"

	"github.com/emc-protocol/edge-matrix-core/core/secrets"
)

const DefaultGRPCPort int = 50000
const DefaultJSONRPCPort int = 50002

// Config is used to parametrize the minimal client
type Config struct {
	GenesisConfig *config.GenesisConfig

	JSONRPC    *JSONRPC
	GRPCAddr   *net.TCPAddr
	LibP2PAddr *net.TCPAddr
	RelayAddr  *net.TCPAddr // the relay address

	PriceLimit         uint64
	MaxAccountEnqueued uint64
	MaxSlots           uint64

	Telemetry   *Telemetry
	EdgeNetwork *network.Config

	DataDir string

	SecretsManager *secrets.SecretsManagerConfig

	LogLevel hclog.Level

	JSONLogFormat bool

	LogFilePath string

	RelayOn        bool
	RelayDiscovery bool

	AppName     string
	AppUrl      string
	AppOrigin   string
	RunningMode string

	EmcHost string
}

// Telemetry holds the config details for metric services
type Telemetry struct {
	PrometheusAddr *net.TCPAddr
}

// JSONRPC holds the config details for the JSON-RPC server
type JSONRPC struct {
	JSONRPCAddr              *net.TCPAddr
	AccessControlAllowOrigin []string
	BatchLengthLimit         uint64
	BlockRangeLimit          uint64
}
