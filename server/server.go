package server

import (
	"errors"
	"fmt"
	cmdConfig "github.com/emc-protocol/edge-matrix-computing/command/server/config"
	"github.com/emc-protocol/edge-matrix-core/core/application"
	"github.com/emc-protocol/edge-matrix-core/core/crypto"
	"github.com/emc-protocol/edge-matrix-core/core/relay"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/emc-protocol/edge-matrix-computing/server/proto"
	"github.com/emc-protocol/edge-matrix-core/core/helper/common"
	"github.com/emc-protocol/edge-matrix-core/core/network"
	"github.com/emc-protocol/edge-matrix-core/core/secrets"
	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

type RunningModeType string

const (
	RunningModeFull RunningModeType = "full"
	RunningModeEdge RunningModeType = "edge"
)
const (
	EdgeDiscProto     = "/disc/1.0"
	EdgeIdentityProto = "/id/1.0"
)

// Server is the central manager of the blockchain client
type Server struct {
	logger hclog.Logger
	config *Config

	// system grpc server
	grpcServer *grpc.Server

	// edge libp2p network
	edgeNetwork *network.Server

	// relay client
	relayClient *relay.RelayClient

	// relay server
	relayServer *relay.RelayServer

	// application syncer Client
	syncAppPeerClient application.SyncAppPeerClient

	// secrets manager
	secretsManager secrets.SecretsManager

	// running mode
	runningMode RunningModeType
}

var dirPaths = []string{
	"blockchain",
	"trie",
}

// newFileLogger returns logger instance that writes all logs to a specified file.
// If log file can't be created, it returns an error
func newFileLogger(config *Config) (hclog.Logger, error) {
	logFileWriter, err := os.Create(config.LogFilePath)
	if err != nil {
		return nil, fmt.Errorf("could not create log file, %w", err)
	}

	return hclog.New(&hclog.LoggerOptions{
		Name:       "edge-matrix",
		Level:      config.LogLevel,
		Output:     logFileWriter,
		JSONFormat: config.JSONLogFormat,
	}), nil
}

// newCLILogger returns minimal logger instance that sends all logs to standard output
func newCLILogger(config *Config) hclog.Logger {
	return hclog.New(&hclog.LoggerOptions{
		Name:       "edge-matrix",
		Level:      config.LogLevel,
		JSONFormat: config.JSONLogFormat,
	})
}

// newLoggerFromConfig creates a new logger which logs to a specified file.
// If log file is not set it outputs to standard output ( console ).
// If log file is specified, and it can't be created the server command will error out
func newLoggerFromConfig(config *Config) (hclog.Logger, error) {
	if config.LogFilePath != "" {
		fileLoggerInstance, err := newFileLogger(config)
		if err != nil {
			return nil, err
		}

		return fileLoggerInstance, nil
	}

	return newCLILogger(config), nil
}

// NewServer creates a new Minimal server, using the passed in configuration
func NewServer(config *Config) (*Server, error) {
	logger, err := newLoggerFromConfig(config)
	if err != nil {
		return nil, fmt.Errorf("could not setup new logger instance, %w", err)
	}

	m := &Server{
		logger:     logger.Named("server"),
		config:     config,
		grpcServer: grpc.NewServer(),
	}

	if m.config.RunningMode == cmdConfig.DefaultRunningMode {
		m.runningMode = RunningModeFull
	} else {
		m.runningMode = RunningModeEdge
	}
	m.logger.Info("Node running", "mode", m.runningMode)

	m.logger.Info("Data dir", "path", config.DataDir)

	// Generate all the paths in the dataDir
	if err := common.SetupDataDir(config.DataDir, dirPaths, 0770); err != nil {
		return nil, fmt.Errorf("failed to create data directories: %w", err)
	}

	// Set up datadog profiler
	if ddErr := m.enableDataDogProfiler(); err != nil {
		m.logger.Error("DataDog profiler setup failed", "err", ddErr.Error())
	}

	// Set up the secrets manager
	if err := m.setupSecretsManager(); err != nil {
		return nil, fmt.Errorf("failed to set up the secrets manager: %w", err)
	}

	if m.runningMode == RunningModeFull {
		// setup edge libp2p network
		edgeNetConfig := config.EdgeNetwork
		edgeNetConfig.DataDir = filepath.Join(m.config.DataDir, "libp2p")
		edgeNetConfig.SecretsManager = m.secretsManager
		edgeNetwork, err := network.NewServer(logger.Named("edge"), edgeNetConfig, EdgeDiscProto, EdgeIdentityProto, true)
		if err != nil {
			return nil, err
		}
		m.edgeNetwork = edgeNetwork
	}

	// setup and start grpc server
	{
		if err := m.setupGRPC(); err != nil {
			return nil, err
		}
	}

	// start network
	{
		if m.runningMode == RunningModeFull {
			// start edge network
			if err := m.edgeNetwork.Start("Edge", m.config.GenesisConfig.Bootnodes); err != nil {
				return nil, err
			}
		}
	}

	{
		// setup edge application
		var endpointHost host.Host

		if m.edgeNetwork != nil {
			endpointHost = m.edgeNetwork.GetHost()
		}

		relayNetConfig := config.EdgeNetwork
		relayNetConfig.DataDir = filepath.Join(m.config.DataDir, "libp2p")
		relayNetConfig.SecretsManager = m.secretsManager

		if m.runningMode == RunningModeEdge {
			// start edge network relay reserv
			relayClient, err := relay.NewRelayClient(logger, relayNetConfig, m.config.RelayOn, m.config.GenesisConfig.Bootnodes)
			if err != nil {
				return nil, err
			}
			endpointHost = relayClient.GetHost()

			m.relayClient = relayClient
			if m.config.RelayOn {
				if err := relayClient.StartRelayReserv(); err != nil {
					return nil, err
				}
			}
		}

		keyBytes, err := m.secretsManager.GetSecret(secrets.ValidatorKey)
		if err != nil {
			return nil, err
		}

		key, err := crypto.BytesToECDSAPrivateKey(keyBytes)
		if err != nil {
			return nil, err
		}

		endpoint, err := application.NewApplicationEndpoint(m.logger, key, endpointHost, m.config.AppName, m.config.AppUrl, m.runningMode == RunningModeEdge)
		if err != nil {
			return nil, err
		}

		if m.runningMode == RunningModeEdge {
			// keep edge peer alive
			err := m.relayClient.StartAlive(endpoint.SubscribeEvents())
			if err != nil {
				return nil, err
			}
		}

		if m.runningMode == RunningModeFull {
			// setup app status syncer
			syncAppclient := application.NewSyncAppPeerClient(m.logger, m.edgeNetwork, m.edgeNetwork.GetHost(), endpoint)
			m.syncAppPeerClient = syncAppclient

			//syncer := application.NewSyncer(
			//	m.logger,
			//	syncAppclient,
			//	application.NewSyncAppPeerService(m.logger, m.edgeNetwork, endpoint, m.blockchain, minerAgent),
			//	m.edgeNetwork.GetHost(),
			//	m.blockchain,
			//	endpoint)
			//// start app status syncer
			//err = syncer.Start(true)
			//if err != nil {
			//	return nil, err
			//}

			// setup and start jsonrpc server
			//if err := m.setupJSONRPC(); err != nil {
			//	return nil, err
			//}

			// start relay server
			if config.RelayAddr.Port > 0 {
				relayListenAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", config.RelayAddr.IP.String(), config.RelayAddr.Port))
				if err != nil {
					return nil, err
				}
				relayServer, err := relay.NewRelayServer(logger, m.secretsManager, relayListenAddr, relayNetConfig, config.RelayDiscovery, m.config.GenesisConfig.Bootnodes)
				if err != nil {
					return nil, err
				}
				logger.Info("LibP2P Relay server running", "addr", relayListenAddr.String()+"/p2p/"+relayServer.GetHost().ID().String())

				err = relayServer.SetupAliveService(syncAppclient)
				if err != nil {
					return nil, fmt.Errorf("unable to setup alive service, %w", err)
				}

				m.relayServer = relayServer

			}
		}
	}

	return m, nil
}

// setupSecretsManager sets up the secrets manager
func (s *Server) setupSecretsManager() error {
	secretsManagerConfig := s.config.SecretsManager
	if secretsManagerConfig == nil {
		// No config provided, use default
		secretsManagerConfig = &secrets.SecretsManagerConfig{
			Type: secrets.Local,
		}
	}

	secretsManagerType := secretsManagerConfig.Type
	secretsManagerParams := &secrets.SecretsManagerParams{
		Logger: s.logger,
	}

	if secretsManagerType == secrets.Local {
		// Only the base directory is required for
		// the local secrets manager
		secretsManagerParams.Extra = map[string]interface{}{
			secrets.Path: s.config.DataDir,
		}
	}

	// Grab the factory method
	secretsManagerFactory, ok := secretsManagerBackends[secretsManagerType]
	if !ok {
		return fmt.Errorf("secrets manager type '%s' not found", secretsManagerType)
	}

	// Instantiate the secrets manager
	secretsManager, factoryErr := secretsManagerFactory(
		secretsManagerConfig,
		secretsManagerParams,
	)

	if factoryErr != nil {
		return fmt.Errorf("unable to instantiate secrets manager, %w", factoryErr)
	}

	s.secretsManager = secretsManager

	return nil
}

// setupGRPC sets up the grpc server and listens on tcp
func (s *Server) setupGRPC() error {
	proto.RegisterSystemServer(s.grpcServer, &systemService{server: s})

	lis, err := net.Listen("tcp", s.config.GRPCAddr.String())
	if err != nil {
		return err
	}

	go func() {
		if err := s.grpcServer.Serve(lis); err != nil {
			s.logger.Error(err.Error())
		}
	}()

	s.logger.Info("GRPC server running", "addr", s.config.GRPCAddr.String())

	return nil
}

// Chain returns the config object of the client
//func (s *Server) Chain() *config.Chain {
//	return s.config
//}

// JoinPeer attempts to add a new peer to the networking server
func (s *Server) JoinPeer(rawPeerMultiaddr string) error {
	return s.edgeNetwork.JoinPeer(rawPeerMultiaddr)
}

// Close closes the Minimal server (blockchain, networking, consensus)
func (s *Server) Close() {
	// Close the networking layer
	if err := s.edgeNetwork.Close(); err != nil {
		s.logger.Error("failed to close networking", "err", err.Error())
	}

	// Close DataDog profiler
	s.closeDataDogProfiler()
}

// Entry is a consensus configuration entry
type Entry struct {
	Enabled bool
	Config  map[string]interface{}
}

func (s *Server) startPrometheusServer(listenAddr *net.TCPAddr) *http.Server {
	srv := &http.Server{
		Addr: listenAddr.String(),
		Handler: promhttp.InstrumentMetricHandler(
			prometheus.DefaultRegisterer, promhttp.HandlerFor(
				prometheus.DefaultGatherer,
				promhttp.HandlerOpts{},
			),
		),
		ReadHeaderTimeout: 60 * time.Second,
	}

	go func() {
		s.logger.Info("Prometheus server started", "addr=", listenAddr.String())

		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("Prometheus HTTP server ListenAndServe", "err", err)
		}
	}()

	return srv
}
