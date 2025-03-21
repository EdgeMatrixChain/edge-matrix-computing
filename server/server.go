package server

import (
	"bufio"
	"errors"
	"fmt"
	appAgent "github.com/EdgeMatrixChain/edge-matrix-computing/agent"
	cmdConfig "github.com/EdgeMatrixChain/edge-matrix-computing/command/server/config"
	"github.com/EdgeMatrixChain/edge-matrix-computing/miner"
	minerProto "github.com/EdgeMatrixChain/edge-matrix-computing/miner/proto"
	"github.com/EdgeMatrixChain/edge-matrix-computing/proxy"
	"github.com/EdgeMatrixChain/edge-matrix-computing/telepool"
	"github.com/EdgeMatrixChain/edge-matrix-computing/versioning"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/application"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/application/proof"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/crypto"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/jsonrpc/web3"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/relay"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/EdgeMatrixChain/edge-matrix-computing/server/proto"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/helper/common"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/network"
	"github.com/EdgeMatrixChain/edge-matrix-core/core/secrets"
	"github.com/hashicorp/go-hclog"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

type RunningModeType string

const DefaultAppBindSyncDuration = 15 * time.Second

const (
	RunningModeFull RunningModeType = "full"
	RunningModeEdge RunningModeType = "edge"
)
const (
	EdgeDiscProto     = "/disc/1.0"
	EdgeIdentityProto = "/id/1.0"
)

// Server is the central manager of the network client
type Server struct {
	logger hclog.Logger
	config *Config

	// jsonrpc stack
	jsonrpcServer *web3.JSONRPC

	// http transparent proxy
	edgeProxyServer *proxy.TransparentProxy

	// system grpc server
	grpcServer *grpc.Server

	// edge libp2p network
	edgeNetwork *network.Server

	// relay client
	relayClient *relay.RelayClient

	// relay server
	relayServer *relay.RelayServer

	// app peers syncer
	appPeerSyncer application.Syncer

	// application syncer Client
	syncAppPeerClient application.SyncAppPeerClient

	// edge matrix app agent
	appAgent *appAgent.AppAgent

	// secrets manager
	secretsManager secrets.SecretsManager

	// running mode
	runningMode RunningModeType

	// telegram pool
	telepool *telepool.TelegramPool

	// edge matrix auth agent
	authAgent *appAgent.AuthAgent
}

func (s *Server) AuthBearer(bearer string, nodeId string, port int) (bool, string) {
	//TODO implement me
	ok, apiKey, err := s.authAgent.AuthBearer(bearer, nodeId, port)
	if err != nil {
		s.logger.Error("AuthBearer failed", "err", err.Error())

		return false, ""
	}
	if !ok {
		s.logger.Warn("AuthBearer failed", "result", ok)
	}

	return true, apiKey
}

func (s *Server) ValidateBearer(bearer string) bool {
	result, err := s.appAgent.ValidateApiKey(bearer)
	if err != nil {
		return false
	}

	return result
}

func (s *Server) GetAppPeer(id string) *application.AppPeer {
	return s.appPeerSyncer.GetAppPeer(id)
}

func (s *Server) GetRelayHost() host.Host {
	return s.relayServer.GetHost()
}

func (s *Server) GetNetworkHost() host.Host {
	return s.edgeNetwork.GetHost()
}

var dirPaths = []string{
	"db",
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

func (s *Server) doAppNodeBind(nodeId string) error {
	err := s.appAgent.BindAppNode(nodeId)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) getAppOrigin() (error, string) {
	err, appOrigin := s.appAgent.GetAppOrigin()
	if err != nil {
		return err, ""
	}
	return nil, appOrigin
}

func (s *Server) GetAppIdl() (error, string) {
	err, appOrigin := s.appAgent.GetAppOrigin()
	if err != nil {
		return err, ""
	}
	return nil, appOrigin
}

func (s *Server) validAppNode(nodeId string) (error, bool) {
	err, bondNodeId := s.appAgent.GetAppNode()
	if err != nil {
		return err, false
	}
	if bondNodeId == nodeId {
		return nil, true
	}
	return nil, false
}

// getAPIKey from http.Request
func getBearer(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

// getForwardInfo from http.Request
func getEdgePath(r *http.Request) *proxy.EdgePath {
	edgePath := &proxy.EdgePath{}

	edgePath.NodeID = r.Header.Get("X-Forwarded-NodeID")
	edgePath.InterfaceURL = r.Header.Get("X-Forwarded-Interface")
	port := r.Header.Get("X-Forwarded-Port")
	atoi, err := strconv.Atoi(port)
	if err == nil {
		edgePath.Port = atoi
	}
	return edgePath
}

// NewServer creates a new Minimal server, using the passed in configuration
func NewServer(config *Config) (*Server, error) {
	logger, logErr := newLoggerFromConfig(config)
	if logErr != nil {
		return nil, fmt.Errorf("could not setup new logger instance, %w", logErr)
	}

	m := &Server{
		logger:     logger.Named("server"),
		config:     config,
		grpcServer: grpc.NewServer(),
		appAgent:   appAgent.NewAppAgent(fmt.Sprintf("%s:%d", config.AppUrl, config.AppPort)),
		authAgent:  appAgent.NewAuthAgent(config.AuthUrl),
	}

	m.logger.Info("Data dir", "path", config.DataDir)

	// Generate all the paths in the dataDir
	if err := common.SetupDataDir(config.DataDir, dirPaths, 0770); err != nil {
		return nil, fmt.Errorf("failed to create data directories: %w", err)
	}

	// Set up datadog profiler
	if err := m.enableDataDogProfiler(); err != nil {
		m.logger.Error("DataDog profiler setup failed", "err", err.Error())
	}

	// Set up the secrets manager
	if err := m.setupSecretsManager(); err != nil {
		return nil, fmt.Errorf("failed to set up the secrets manager: %w", err)
	}

	var endpointHost host.Host

	if m.config.RunningMode == cmdConfig.DefaultRunningMode {
		m.runningMode = RunningModeFull
	} else {
		m.runningMode = RunningModeEdge
	}
	m.logger.Info("Node running", "mode", m.runningMode)

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
		endpointHost = m.edgeNetwork.GetHost()

		// start edge network
		if err = m.edgeNetwork.Start("Edge", m.config.GenesisConfig.Bootnodes); err != nil {
			return nil, err
		}
	} else {
		relayNetConfig := config.EdgeNetwork
		relayNetConfig.DataDir = filepath.Join(m.config.DataDir, "libp2p")
		relayNetConfig.SecretsManager = m.secretsManager

		// start edge network relay reserv
		relayClient, rlyErr := relay.NewRelayClient(logger, relayNetConfig, m.config.RelayOn, m.config.GenesisConfig.Relaynodes)
		if rlyErr != nil {
			return nil, rlyErr
		}
		endpointHost = relayClient.GetHost()

		m.relayClient = relayClient
		if m.config.RelayOn {
			if err := relayClient.StartRelayReserv(); err != nil {
				return nil, err
			}
		}
	}

	{
		// setup edge application
		keyBytes, keyBytesErr := m.secretsManager.GetSecret(secrets.ValidatorKey)
		if keyBytesErr != nil {
			return nil, keyBytesErr
		}

		key, keyErr := crypto.BytesToECDSAPrivateKey(keyBytes)
		if keyErr != nil {
			return nil, keyErr
		}

		endpoint, endpointErr := application.NewApplicationEndpoint(m.logger, key, endpointHost, m.config.AppName, m.config.AppUrl, m.config.AppPort, versioning.Version)
		if endpointErr != nil {
			return nil, endpointErr
		}

		endpoint.SetSigner(proof.NewEIP155Signer(crypto.AllForksEnabled.At(0), uint64(m.config.GenesisConfig.NetworkId)))

		// bind app agent
		if !m.config.AppNoAgent {
			if err := m.doAppNodeBind(endpointHost.ID().String()); err != nil {
				m.logger.Error("doAppNodeBind", "err", err.Error())
			}

			appOriginErr, appOrigin := m.getAppOrigin()
			if appOriginErr != nil {
				m.logger.Error("getAppOrigin", "err", appOriginErr.Error())
			}
			endpoint.SetAppOrigin(appOrigin)
		}

		endpoint.AddHandler("/alive", func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			resp := fmt.Sprintf("{\"time\":\"%s\"}", time.Now().String())
			w.Write([]byte(resp))
		})

		endpoint.AddHandler("/idl", func(w http.ResponseWriter, r *http.Request) {
			defer r.Body.Close()
			if m.config.AppNoAgent {
				idlData, err := os.ReadFile("idl.json")
				if nil != err {
					idlData = []byte("[]")
				}
				application.WriteSignedResponse(w, idlData, endpoint)

				return
			}

			appIdlErr, appIdl := m.GetAppIdl()
			if appIdlErr != nil {
				application.WriteSignedResponse(w, []byte("[]"), endpoint)

				return
			}

			if len(appIdl) > 0 {
				application.WriteSignedResponse(w, []byte(appIdl), endpoint)
			} else {
				application.WriteSignedResponse(w, []byte("[]"), endpoint)
			}

		})

		endpoint.AddHandler(proxy.TransparentForwardUrl, func(w http.ResponseWriter, r *http.Request) {
			m.logger.Debug(proxy.TransparentForwardUrl, "RemoteAddr", r.RemoteAddr, "Host", r.Host)

			if !m.config.AppNoAuth && !m.ValidateBearer(getBearer(r)) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)

				return
			}

			defer r.Body.Close()

			//m.logger.Debug(proxy.TransparentForwardUrl, "body", string(body))
			edgePath := getEdgePath(r)

			client := &http.Client{}
			var targetURL = ""
			if m.config.AppNoAgent {
				targetURL = fmt.Sprintf("%s:%d/%s", m.config.AppUrl, edgePath.Port, edgePath.InterfaceURL)
			} else {
				err, proxyPath := m.appAgent.GetProxyPath()
				if err != nil {
					http.Error(w, fmt.Sprintf("%s %s", proxy.TransparentForwardUrl, err.Error()), http.StatusServiceUnavailable)

					return
				}
				targetURL = fmt.Sprintf("%s:%d%s/%d/%s", m.config.AppUrl, m.config.AppPort, proxyPath, edgePath.Port, edgePath.InterfaceURL)
			}
			m.logger.Debug(proxy.TransparentForwardUrl, "targetURL", targetURL)

			req, reqErr := http.NewRequest(r.Method, targetURL, r.Body)
			if reqErr != nil {
				http.Error(w, fmt.Sprintf("%s %s", proxy.TransparentForwardUrl, reqErr.Error()), http.StatusInternalServerError)

				return
			}

			for key, values := range r.Header {
				for _, value := range values {
					req.Header.Add(key, value)
					m.logger.Debug(proxy.TransparentForwardUrl, key, value)
				}
			}

			resp, respErr := client.Do(req)
			if respErr != nil {
				http.Error(w, "Failed to connect to target server", http.StatusBadGateway)

				return
			}
			defer resp.Body.Close()

			for key, values := range resp.Header {
				for _, value := range values {
					w.Header().Add(key, value)
				}
			}
			w.WriteHeader(resp.StatusCode)
			if resp.Header.Get("Content-Type") == "text/event-stream" {
				reader := bufio.NewReader(resp.Body)
				for {
					line, lineErr := reader.ReadBytes('\n')
					if lineErr != nil {
						if lineErr == io.EOF {

							return
						}
						m.logger.Warn(proxy.TransparentForwardUrl, "err", fmt.Sprintf("Error reading SSE stream: %v\n", lineErr))

						return
					}

					_, writeErr := w.Write(line)
					if writeErr != nil {
						m.logger.Warn(proxy.TransparentForwardUrl, "err", fmt.Sprintf("Error writing to client: %v\n", writeErr))

						return
					}

					w.(http.Flusher).Flush()
				}
			} else {
				io.Copy(w, resp.Body)
			}
		})

		if m.runningMode == RunningModeFull {
			// setup app status syncer
			syncAppclient := application.NewSyncAppPeerClient(m.logger, m.edgeNetwork, m.edgeNetwork.GetHost(), endpoint)
			m.syncAppPeerClient = syncAppclient

			syncer := application.NewSyncer(
				m.logger,
				syncAppclient,
				application.NewSyncAppPeerService(m.logger, m.edgeNetwork, endpoint),
				m.edgeNetwork.GetHost(),
				endpoint)
			// start app status syncer
			if err := syncer.Start(true); err != nil {
				return nil, err
			}
			m.appPeerSyncer = syncer

			// Setup telegram pool
			m.telepool = telepool.NewTelegramPool(
				logger,
				&telepool.Config{
					MaxSlots:           m.config.MaxSlots,
					MaxAccountEnqueued: m.config.MaxAccountEnqueued,
				},
				m,
				telepool.NewEIP155Signer(crypto.AllForksEnabled.At(0), uint64(m.config.GenesisConfig.NetworkId)),
			)

			// setup and start jsonrpc server
			if err := m.setupJSONRPC(); err != nil {
				return nil, err
			}

			// setup and start transparent proxy server
			if err := m.setupTransparentProxy(); err != nil {
				return nil, err
			}

			// start relay server
			if config.RelayAddr.Port > 0 {
				relayListenAddr, relayListenAddrErr := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", config.RelayAddr.IP.String(), config.RelayAddr.Port))
				if relayListenAddrErr != nil {
					return nil, relayListenAddrErr
				}
				relayServer, relayServerErr := relay.NewRelayServer(logger, m.secretsManager, relayListenAddr, config.TransparentProxy.ProxyAddr, syncAppclient, config.RelayDiscovery, m.config.GenesisConfig.Relaynodes)
				if relayServerErr != nil {
					return nil, relayServerErr
				}
				logger.Info("LibP2P Relay server running", "addr", relayListenAddr.String()+"/p2p/"+relayServer.GetHost().ID().String())

				if err := relayServer.SetupAliveService(syncAppclient); err != nil {
					return nil, fmt.Errorf("unable to setup alive service, %w", err)
				}

				m.relayServer = relayServer

			}
		} else {
			// keep edge peer alive
			if err := m.relayClient.StartAlive(endpoint.SubscribeEvents()); err != nil {
				return nil, err
			}

			// do agent binding
			if !m.config.AppNoAgent {
				go func() {
					ticker := time.NewTicker(DefaultAppBindSyncDuration)
					defer ticker.Stop()
					for {
						<-ticker.C

						if err := m.doAppNodeBind(endpointHost.ID().String()); err != nil {
							m.logger.Error("doAppNodeBind", "err", err.Error())
						}

						appOriginErr, appOrigin := m.getAppOrigin()
						if appOriginErr != nil {
							m.logger.Error("getAppOrigin", "err", appOriginErr.Error())
						}
						endpoint.SetAppOrigin(appOrigin)

						m.logger.Info("binding", "NodeID", endpointHost.ID().String(), "AppOrigin", appOrigin)
					}
				}()
			}
		}

		// init miner grpc service
		minerAgent := miner.NewMinerHubAgent(m.logger, m.secretsManager)
		if _, err := m.initMinerService(minerAgent, endpointHost, m.secretsManager); err != nil {
			return nil, err
		}

		// setup and start grpc server
		{
			if err := m.setupGRPC(); err != nil {
				return nil, err
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

// setupJSONRCP sets up the JSONRPC server, using the set configuration
func (s *Server) setupJSONRPC() error {
	hub := &jsonRPCHub{
		TelegramPool:      s.telepool,
		Server:            s.edgeNetwork,
		SyncAppPeerClient: s.syncAppPeerClient,
	}
	conf := &web3.Config{
		Store:                    hub,
		Addr:                     s.config.JSONRPC.JSONRPCAddr,
		NetworkID:                uint64(s.config.GenesisConfig.NetworkId),
		NetworkName:              s.config.GenesisConfig.Name,
		Version:                  versioning.Version,
		AccessControlAllowOrigin: s.config.JSONRPC.AccessControlAllowOrigin,
	}

	srv, err := web3.NewJSONRPC(s.logger, conf)
	if err != nil {
		return err
	}

	s.jsonrpcServer = srv

	return nil
}

// setupTransparentProxy sets up the edge transparent proxy server, using the set configuration
func (s *Server) setupTransparentProxy() error {
	conf := &proxy.Config{
		Store:                    s,
		Addr:                     s.config.TransparentProxy.ProxyAddr,
		NetworkID:                uint64(s.config.GenesisConfig.NetworkId),
		NetworkName:              s.config.GenesisConfig.Name,
		Version:                  versioning.Version,
		AccessControlAllowOrigin: s.config.TransparentProxy.AccessControlAllowOrigin,
	}

	srv, err := proxy.NewTransportProxy(s.logger, conf, s.config.AppNoAuth)
	if err != nil {
		return err
	}

	s.edgeProxyServer = srv

	return nil
}

// initMinerService sets up the Miner grpc service
func (s *Server) initMinerService(minerAgent *miner.MinerHubAgent, host host.Host, secretsManager secrets.SecretsManager) (*miner.MinerService, error) {
	if s.grpcServer != nil {
		minerService := miner.NewMinerService(s.logger, minerAgent, host, secretsManager)
		minerProto.RegisterMinerServer(s.grpcServer, minerService)
		return minerService, nil
	}

	return nil, errors.New("grpcServer is nil")
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
	if s.edgeNetwork != nil {
		if err := s.edgeNetwork.Close(); err != nil {
			s.logger.Error("failed to close networking", "err", err.Error())
		}
	}
	// Close the syncAppPeerClient
	if s.syncAppPeerClient != nil {
		s.syncAppPeerClient.Close()
	}

	// close the relayClient
	if s.relayClient != nil {
		s.relayClient.Close()
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

type jsonRPCHub struct {
	*telepool.TelegramPool
	*network.Server
	application.SyncAppPeerClient
}

func (j *jsonRPCHub) GetPeers() int {
	return len(j.Server.Peers())
}
