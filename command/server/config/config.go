package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/EdgeMatrixChain/edge-matrix-core/core/network"
	"github.com/hashicorp/hcl"
	"gopkg.in/yaml.v3"
)

// Config defines the server configuration params
type Config struct {
	GenesisPath              string     `json:"chain_config" yaml:"chain_config"`
	SecretsConfigPath        string     `json:"secrets_config" yaml:"secrets_config"`
	DataDir                  string     `json:"data_dir" yaml:"data_dir"`
	GRPCAddr                 string     `json:"grpc_addr" yaml:"grpc_addr"`
	JSONRPCAddr              string     `json:"jsonrpc_addr" yaml:"jsonrpc_addr"`
	TransparentProxyAddr     string     `json:"transparent_proxy_addr" yaml:"transparent_proxy_addr"`
	Telemetry                *Telemetry `json:"telemetry" yaml:"telemetry"`
	Network                  *Network   `json:"network" yaml:"network"`
	TelePool                 *TelePool  `json:"tele_pool" yaml:"tele_pool"`
	LogLevel                 string     `json:"log_level" yaml:"log_level"`
	Headers                  *Headers   `json:"headers" yaml:"headers"`
	LogFilePath              string     `json:"log_to" yaml:"log_to"`
	JSONRPCBatchRequestLimit uint64     `json:"json_rpc_batch_request_limit" yaml:"json_rpc_batch_request_limit"`
	JSONRPCBlockRangeLimit   uint64     `json:"json_rpc_block_range_limit" yaml:"json_rpc_block_range_limit"`
	JSONLogFormat            bool       `json:"json_log_format" yaml:"json_log_format"`

	NumBlockConfirmations uint64 `json:"num_block_confirmations" yaml:"num_block_confirmations"`

	RelayOn        bool   `json:"relay_on,omitempty" yaml:"relay_on,omitempty"`
	RelayDiscovery bool   `json:"relay_discovery,omitempty" yaml:"relay_discovery,omitempty"`
	RunningMode    string `json:"running_mode,omitempty" yaml:"running_mode,omitempty"`

	AppUrl     string `json:"app_url,omitempty" yaml:"app_url,omitempty"`
	AppPort    uint64 `json:"app_port,omitempty" yaml:"app_port,omitempty"`
	AppName    string `json:"app_name,omitempty" yaml:"app_name,omitempty"`
	AppNoAuth  bool   `json:"app_no_auth,omitempty" yaml:"app_no_auth,omitempty"`
	AppNoAgent bool   `json:"app_no_agent,omitempty" yaml:"app_no_agent,omitempty"`

	AuthUrl string `json:"auth_url,omitempty" yaml:"auth_url,omitempty"`
}

// Telemetry holds the config details for metric services.
type Telemetry struct {
	PrometheusAddr string `json:"prometheus_addr" yaml:"prometheus_addr"`
}

// Network defines the network configuration params
type Network struct {
	NoDiscover       bool   `json:"no_discover" yaml:"no_discover"`
	Libp2pAddr       string `json:"libp2p_addr" yaml:"libp2p_addr"`
	EdgeLibp2pAddr   string `json:"edge_libp2p_addr" yaml:"libp2p_addr"`
	RelayLibp2pAddr  string `json:"relay_libp2p_addr" yaml:"libp2p_addr"`
	NatAddr          string `json:"nat_addr" yaml:"nat_addr"`
	DNSAddr          string `json:"dns_addr" yaml:"dns_addr"`
	MaxPeers         int64  `json:"max_peers,omitempty" yaml:"max_peers,omitempty"`
	MaxOutboundPeers int64  `json:"max_outbound_peers,omitempty" yaml:"max_outbound_peers,omitempty"`
	MaxInboundPeers  int64  `json:"max_inbound_peers,omitempty" yaml:"max_inbound_peers,omitempty"`
}

// TelePool defines the TelePool configuration params
type TelePool struct {
	MaxSlots           uint64 `json:"max_slots" yaml:"max_slots"`
	MaxAccountEnqueued uint64 `json:"max_account_enqueued" yaml:"max_account_enqueued"`
}

// Headers defines the HTTP response headers required to enable CORS.
type Headers struct {
	AccessControlAllowOrigins []string `json:"access_control_allow_origins" yaml:"access_control_allow_origins"`
}

const (
	// DefaultJSONRPCBatchRequestLimit maximum length allowed for json_rpc batch requests
	DefaultJSONRPCBatchRequestLimit uint64 = 20

	// DefaultJSONRPCBlockRangeLimit maximum block range allowed for json_rpc
	// requests with fromBlock/toBlock values (e.g. eth_getLogs)
	DefaultJSONRPCBlockRangeLimit uint64 = 1000

	DefaultRunningMode string = "full"
)

// DefaultConfig returns the default server configuration
func DefaultConfig() *Config {
	defaultNetworkConfig := network.DefaultConfig()
	return &Config{
		GenesisPath: "./genesis.json",
		DataDir:     "",
		Network: &Network{
			NoDiscover:       defaultNetworkConfig.NoDiscover,
			MaxPeers:         defaultNetworkConfig.MaxPeers,
			MaxOutboundPeers: defaultNetworkConfig.MaxOutboundPeers,
			MaxInboundPeers:  defaultNetworkConfig.MaxInboundPeers,
			Libp2pAddr: fmt.Sprintf("%s:%d",
				defaultNetworkConfig.Addr.IP,
				defaultNetworkConfig.Addr.Port,
			),
			EdgeLibp2pAddr: fmt.Sprintf("%s:%d",
				defaultNetworkConfig.Addr.IP,
				network.DefaultEdgeLibp2pPort,
			),
			RelayLibp2pAddr: fmt.Sprintf("%s:%d",
				defaultNetworkConfig.Addr.IP,
				network.DefaultRelayLibp2pPort,
			),
		},
		Telemetry: &Telemetry{},
		TelePool: &TelePool{
			MaxSlots:           4096,
			MaxAccountEnqueued: 128,
		},
		LogLevel: "INFO",
		Headers: &Headers{
			AccessControlAllowOrigins: []string{"*"},
		},
		LogFilePath:              "",
		JSONRPCBatchRequestLimit: DefaultJSONRPCBatchRequestLimit,
		JSONRPCBlockRangeLimit:   DefaultJSONRPCBlockRangeLimit,
		RelayOn:                  false,
		RelayDiscovery:           false,
		RunningMode:              DefaultRunningMode,
	}
}

// ReadConfigFile reads the config file from the specified path, builds a Config object
// and returns it.
//
// Supported file types: .json, .hcl, .yaml, .yml
func ReadConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var unmarshalFunc func([]byte, interface{}) error

	switch {
	case strings.HasSuffix(path, ".hcl"):
		unmarshalFunc = hcl.Unmarshal
	case strings.HasSuffix(path, ".json"):
		unmarshalFunc = json.Unmarshal
	case strings.HasSuffix(path, ".yaml"), strings.HasSuffix(path, ".yml"):
		unmarshalFunc = yaml.Unmarshal
	default:
		return nil, fmt.Errorf("suffix of %s is neither hcl, json, yaml nor yml", path)
	}

	config := DefaultConfig()
	config.Network = new(Network)
	config.Network.MaxPeers = -1
	config.Network.MaxInboundPeers = -1
	config.Network.MaxOutboundPeers = -1

	if err := unmarshalFunc(data, config); err != nil {
		return nil, err
	}

	return config, nil
}
