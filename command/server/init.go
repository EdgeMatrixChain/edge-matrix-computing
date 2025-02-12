package server

import (
	"errors"
	"fmt"
	"github.com/emc-protocol/edge-matrix-computing/config"
	"math"
	"net"

	serverConfig "github.com/emc-protocol/edge-matrix-computing/command/server/config"

	"github.com/emc-protocol/edge-matrix-core/core/network/common"

	"github.com/emc-protocol/edge-matrix-computing/command/helper"
	"github.com/emc-protocol/edge-matrix-core/core/network"
	"github.com/emc-protocol/edge-matrix-core/core/secrets"
)

var (
	errInvalidBlockTime       = errors.New("invalid block time specified")
	errDataDirectoryUndefined = errors.New("data directory not defined")
	errMinerCanisterUndefined = errors.New("miner canister not defined")
)

func (p *serverParams) initConfigFromFile() error {
	var parseErr error

	if p.rawConfig, parseErr = serverConfig.ReadConfigFile(p.configPath); parseErr != nil {
		return parseErr
	}

	return nil
}

func (p *serverParams) initRawParams() error {
	if err := p.initSecretsConfig(); err != nil {
		return err
	}

	if err := p.initGenesisConfig(); err != nil {
		return err
	}

	if err := p.initDataDirLocation(); err != nil {
		return err
	}

	p.initPeerLimits()
	p.initLogFileLocation()

	return p.initAddresses()
}

func (p *serverParams) initDataDirLocation() error {
	if p.rawConfig.DataDir == "" {
		return errDataDirectoryUndefined
	}

	return nil
}

func (p *serverParams) initLogFileLocation() {
	if p.isLogFileLocationSet() {
		p.logFileLocation = p.rawConfig.LogFilePath
	}
}

func (p *serverParams) initSecretsConfig() error {
	if !p.isSecretsConfigPathSet() {
		return nil
	}

	var parseErr error

	if p.secretsConfig, parseErr = secrets.ReadConfig(
		p.rawConfig.SecretsConfigPath,
	); parseErr != nil {
		return fmt.Errorf("unable to read secrets config file, %w", parseErr)
	}

	return nil
}

func (p *serverParams) initGenesisConfig() error {
	var parseErr error

	if p.genesisConfig, parseErr = config.Import(
		p.rawConfig.GenesisPath,
	); parseErr != nil {
		return parseErr
	}

	return nil
}

//func (p *serverParams) initDevMode() {
//	// Dev mode:
//	// - disables peer discovery
//	// - enables all forks
//	p.rawConfig.Network.NoDiscover = true
//	p.genesisConfig.Params.Forks = config.AllForksEnabled
//
//	p.initDevConsensusConfig()
//}

//func (p *serverParams) initDevConsensusConfig() {
//	if !p.isDevConsensus() {
//		return
//	}
//
//	p.genesisConfig.Params.Engine = map[string]interface{}{
//		string(server.DevConsensus): map[string]interface{}{
//			"interval": p.devInterval,
//		},
//	}
//}

func (p *serverParams) initPeerLimits() {
	if !p.isMaxPeersSet() && !p.isPeerRangeSet() {
		// No peer limits specified, use the default limits
		p.initDefaultPeerLimits()

		return
	}

	if p.isPeerRangeSet() {
		// Some part of the peer range is specified
		p.initUsingPeerRange()

		return
	}

	if p.isMaxPeersSet() {
		// The max peer value is specified, derive precise limits
		p.initUsingMaxPeers()

		return
	}
}

func (p *serverParams) initDefaultPeerLimits() {
	defaultNetworkConfig := network.DefaultConfig()

	p.rawConfig.Network.MaxInboundPeers = defaultNetworkConfig.MaxInboundPeers
	p.rawConfig.Network.MaxOutboundPeers = defaultNetworkConfig.MaxOutboundPeers
}

func (p *serverParams) initUsingPeerRange() {
	defaultConfig := network.DefaultConfig()

	if p.rawConfig.Network.MaxInboundPeers == unsetPeersValue {
		p.rawConfig.Network.MaxInboundPeers = defaultConfig.MaxInboundPeers
	}

	if p.rawConfig.Network.MaxOutboundPeers == unsetPeersValue {
		p.rawConfig.Network.MaxOutboundPeers = defaultConfig.MaxOutboundPeers
	}

	p.rawConfig.Network.MaxPeers = p.rawConfig.Network.MaxInboundPeers + p.rawConfig.Network.MaxOutboundPeers
}

func (p *serverParams) initUsingMaxPeers() {
	p.rawConfig.Network.MaxOutboundPeers = int64(
		math.Floor(
			float64(p.rawConfig.Network.MaxPeers) * network.DefaultDialRatio,
		),
	)
	p.rawConfig.Network.MaxInboundPeers = p.rawConfig.Network.MaxPeers - p.rawConfig.Network.MaxOutboundPeers
}

func (p *serverParams) initAddresses() error {
	if err := p.initPrometheusAddress(); err != nil {
		return err
	}

	if err := p.initLibp2pAddress(); err != nil {
		return err
	}

	if err := p.initNATAddress(); err != nil {
		return err
	}

	if err := p.initDNSAddress(); err != nil {
		return err
	}

	if err := p.initJSONRPCAddress(); err != nil {
		return err
	}

	return p.initGRPCAddress()
}

func (p *serverParams) initPrometheusAddress() error {
	if !p.isPrometheusAddressSet() {
		return nil
	}

	var parseErr error

	if p.prometheusAddress, parseErr = helper.ResolveAddr(
		p.rawConfig.Telemetry.PrometheusAddr,
		helper.AllInterfacesBinding,
	); parseErr != nil {
		return parseErr
	}

	return nil
}

func (p *serverParams) initLibp2pAddress() error {
	var parseErr error

	if p.libp2pAddress, parseErr = helper.ResolveAddr(
		p.rawConfig.Network.Libp2pAddr,
		helper.LocalHostBinding,
	); parseErr != nil {
		return parseErr
	}

	if p.edgeLibp2pAddress, parseErr = helper.ResolveAddr(
		p.rawConfig.Network.EdgeLibp2pAddr,
		helper.LocalHostBinding,
	); parseErr != nil {
		return parseErr
	}

	if p.relayLibp2pAddress, parseErr = helper.ResolveAddr(
		p.rawConfig.Network.RelayLibp2pAddr,
		helper.LocalHostBinding,
	); parseErr != nil {
		return parseErr
	}

	return nil
}

func (p *serverParams) initNATAddress() error {
	if !p.isNATAddressSet() {
		return nil
	}

	if p.natAddress = net.ParseIP(
		p.rawConfig.Network.NatAddr,
	); p.natAddress == nil {
		return errInvalidNATAddress
	}

	return nil
}

func (p *serverParams) initDNSAddress() error {
	if !p.isDNSAddressSet() {
		return nil
	}

	var parseErr error

	if p.dnsAddress, parseErr = common.MultiAddrFromDNS(
		p.rawConfig.Network.DNSAddr, p.libp2pAddress.Port,
	); parseErr != nil {
		return parseErr
	}

	return nil
}

func (p *serverParams) initJSONRPCAddress() error {
	var parseErr error

	if p.jsonRPCAddress, parseErr = helper.ResolveAddr(
		p.rawConfig.JSONRPCAddr,
		helper.AllInterfacesBinding,
	); parseErr != nil {
		return parseErr
	}

	return nil
}

func (p *serverParams) initGRPCAddress() error {
	var parseErr error

	if p.grpcAddress, parseErr = helper.ResolveAddr(
		p.rawConfig.GRPCAddr,
		helper.LocalHostBinding,
	); parseErr != nil {
		return parseErr
	}

	return nil
}
