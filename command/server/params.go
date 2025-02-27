package server

import (
	"errors"
	config2 "github.com/EdgeMatrixChain/edge-matrix-computing/config"
	"net"

	"github.com/EdgeMatrixChain/edge-matrix-computing/command/server/config"
	"github.com/EdgeMatrixChain/edge-matrix-computing/server"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/network"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/secrets"
	"github.com/hashicorp/go-hclog"
	"github.com/multiformats/go-multiaddr"
)

const (
	configFlag                   = "config"
	genesisPathFlag              = "network"
	dataDirFlag                  = "data-dir"
	edgeLibp2pAddressFlag        = "libp2p"
	relayLibp2pAddressFlag       = "relay-libp2p"
	prometheusAddressFlag        = "prometheus"
	natFlag                      = "nat"
	dnsFlag                      = "dns"
	maxPeersFlag                 = "max-peers"
	maxInboundPeersFlag          = "max-inbound-peers"
	maxOutboundPeersFlag         = "max-outbound-peers"
	jsonRPCBatchRequestLimitFlag = "json-rpc-batch-request-limit"
	jsonRPCBlockRangeLimitFlag   = "json-rpc-block-range-limit"
	maxSlotsFlag                 = "max-slots"
	maxEnqueuedFlag              = "max-enqueued"
	secretsConfigFlag            = "secrets-config"
	devIntervalFlag              = "dev-interval"
	devFlag                      = "dev"
	corsOriginFlag               = "access-control-allow-origins"
	logFileLocationFlag          = "log-to"

	relayOnFlag        = "relay-on"
	relayDiscoveryFlag = "relay-discovery"

	runningModeFlag = "running-mode"
	appNameFlag     = "app-name"
	appUrlFlag      = "app-url"
	appPortFlag     = "app-default-port"
	appNoAuthFlag   = "app-no-auth"
	appNoAgentFlag  = "app-no-agent"
)

const (
	unsetPeersValue = -1
)

var (
	params = &serverParams{
		rawConfig: &config.Config{
			Telemetry: &config.Telemetry{},
			Network:   &config.Network{},
			TelePool:  &config.TelePool{},
		},
	}
)

var (
	errInvalidNATAddress = errors.New("could not parse NAT IP address")
)

type serverParams struct {
	rawConfig  *config.Config
	configPath string

	libp2pAddress           *net.TCPAddr
	edgeLibp2pAddress       *net.TCPAddr
	relayLibp2pAddress      *net.TCPAddr
	prometheusAddress       *net.TCPAddr
	natAddress              net.IP
	dnsAddress              multiaddr.Multiaddr
	grpcAddress             *net.TCPAddr
	jsonRPCAddress          *net.TCPAddr
	transparentProxyAddress *net.TCPAddr

	devInterval uint64
	isDevMode   bool

	corsAllowedOrigins []string

	genesisConfig *config2.GenesisConfig
	secretsConfig *secrets.SecretsManagerConfig

	logFileLocation string
}

func (p *serverParams) isMaxPeersSet() bool {
	return p.rawConfig.Network.MaxPeers != unsetPeersValue
}

func (p *serverParams) isPeerRangeSet() bool {
	return p.rawConfig.Network.MaxInboundPeers != unsetPeersValue ||
		p.rawConfig.Network.MaxOutboundPeers != unsetPeersValue
}

func (p *serverParams) isSecretsConfigPathSet() bool {
	return p.rawConfig.SecretsConfigPath != ""
}

func (p *serverParams) isPrometheusAddressSet() bool {
	return p.rawConfig.Telemetry.PrometheusAddr != ""
}

func (p *serverParams) isNATAddressSet() bool {
	return p.rawConfig.Network.NatAddr != ""
}

func (p *serverParams) isDNSAddressSet() bool {
	return p.rawConfig.Network.DNSAddr != ""
}

func (p *serverParams) isLogFileLocationSet() bool {
	return p.rawConfig.LogFilePath != ""
}

func (p *serverParams) setRawGRPCAddress(grpcAddress string) {
	p.rawConfig.GRPCAddr = grpcAddress
}

func (p *serverParams) setRawJSONRPCAddress(jsonRPCAddress string) {
	p.rawConfig.JSONRPCAddr = jsonRPCAddress
}

func (p *serverParams) setRawTransparentProxyAddress(transparentProxyAddress string) {
	p.rawConfig.TransparentProxyAddr = transparentProxyAddress
}

func (p *serverParams) setJSONLogFormat(jsonLogFormat bool) {
	p.rawConfig.JSONLogFormat = jsonLogFormat
}

func (p *serverParams) generateConfig() *server.Config {
	return &server.Config{
		GenesisConfig: p.genesisConfig,
		TransparentProxy: &server.TransparentProxyConfig{
			ProxyAddr:                p.transparentProxyAddress,
			AccessControlAllowOrigin: p.corsAllowedOrigins,
		},
		JSONRPC: &server.JSONRPC{
			JSONRPCAddr:              p.jsonRPCAddress,
			AccessControlAllowOrigin: p.corsAllowedOrigins,
			BatchLengthLimit:         p.rawConfig.JSONRPCBatchRequestLimit,
			BlockRangeLimit:          p.rawConfig.JSONRPCBlockRangeLimit,
		},
		GRPCAddr:   p.grpcAddress,
		LibP2PAddr: p.libp2pAddress,
		Telemetry: &server.Telemetry{
			PrometheusAddr: p.prometheusAddress,
		},
		EdgeNetwork: &network.Config{
			NoDiscover:       p.rawConfig.Network.NoDiscover,
			Addr:             p.edgeLibp2pAddress,
			NatAddr:          p.natAddress,
			DNS:              p.dnsAddress,
			DataDir:          p.rawConfig.DataDir,
			MaxPeers:         p.rawConfig.Network.MaxPeers,
			MaxInboundPeers:  p.rawConfig.Network.MaxInboundPeers,
			MaxOutboundPeers: p.rawConfig.Network.MaxOutboundPeers,
			NetworkID:        p.genesisConfig.NetworkId,
			//Config:            p.genesisConfig,
		},
		RelayAddr:          p.relayLibp2pAddress,
		DataDir:            p.rawConfig.DataDir,
		MaxSlots:           p.rawConfig.TelePool.MaxSlots,
		MaxAccountEnqueued: p.rawConfig.TelePool.MaxAccountEnqueued,
		SecretsManager:     p.secretsConfig,
		LogLevel:           hclog.LevelFromString(p.rawConfig.LogLevel),
		JSONLogFormat:      p.rawConfig.JSONLogFormat,
		LogFilePath:        p.logFileLocation,

		RelayOn:        p.rawConfig.RelayOn,
		RelayDiscovery: p.rawConfig.RelayDiscovery,

		RunningMode: p.rawConfig.RunningMode,
		AppName:     p.rawConfig.AppName,
		AppUrl:      p.rawConfig.AppUrl,
		AppPort:     p.rawConfig.AppPort,
		AppNoAuth:   p.rawConfig.AppNoAuth,
		AppNoAgent:  p.rawConfig.AppNoAgent,

		EmcHost: p.rawConfig.EmcHost,
	}
}
