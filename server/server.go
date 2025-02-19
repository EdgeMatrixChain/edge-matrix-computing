package server

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	appAgent "github.com/emc-protocol/edge-matrix-computing/agent"
	cmdConfig "github.com/emc-protocol/edge-matrix-computing/command/server/config"
	"github.com/emc-protocol/edge-matrix-computing/proxy"
	"github.com/emc-protocol/edge-matrix-core/core/application"
	"github.com/emc-protocol/edge-matrix-core/core/application/proof"
	"github.com/emc-protocol/edge-matrix-core/core/crypto"
	"github.com/emc-protocol/edge-matrix-core/core/jsonrpc/web3"
	"github.com/emc-protocol/edge-matrix-core/core/relay"
	"github.com/emc-protocol/edge-matrix-core/core/types"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

const DefaultAppBindSyncDuration = 15 * time.Second

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
}

func (s *Server) ValidateBearer(bearer string) bool {
	//TODO implement ValidateBearer
	err := s.appAgent.ValidateApiKey(bearer)
	if err != nil {
		return false
	}

	return true
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
		appAgent:   appAgent.NewAppAgent(fmt.Sprintf("%s:%d", config.AppUrl, config.AppPort)),
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
			relayClient, err := relay.NewRelayClient(logger, relayNetConfig, m.config.RelayOn, m.config.GenesisConfig.Relaynodes)
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

		endpoint, err := application.NewApplicationEndpoint(m.logger, key, endpointHost, m.config.AppName, m.config.AppUrl, m.config.AppPort, m.runningMode == RunningModeEdge)
		if err != nil {
			return nil, err
		}

		endpoint.SetSigner(proof.NewEIP155Signer(proof.AllForksEnabled.At(0), uint64(m.config.GenesisConfig.NetworkId)))

		// bind app agent
		if !m.config.AppNoAgent {
			err = m.doAppNodeBind(endpointHost.ID().String())
			if err != nil {
				m.logger.Error("doAppNodeBind", "err", err.Error())
			}

			err, appOrigin := m.getAppOrigin()
			if err != nil {
				m.logger.Error("getAppOrigin", "err", err.Error())
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
			err, appIdl := m.GetAppIdl()
			if err != nil {
				// Fetch idl json text through GET #{appUrl}/getAppIdl
				idlData, err := os.ReadFile("idl.json")
				if nil != err {
					idlData = []byte("[]")
				}
				application.WriteSignedResponse(w, idlData, endpoint)
			} else {
				if len(appIdl) > 0 {
					application.WriteSignedResponse(w, []byte(appIdl), endpoint)
				} else {
					application.WriteSignedResponse(w, []byte("[]"), endpoint)
				}
			}
		})

		endpoint.AddHandler(proxy.TransparentForwardUrl, func(w http.ResponseWriter, r *http.Request) {
			m.logger.Debug(proxy.TransparentForwardUrl, "RemoteAddr", r.RemoteAddr, "Host", r.Host)

			bearer := getBearer(r)
			if bearer == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)

				return
			}

			if !m.config.AppNoAuth && !m.ValidateBearer(bearer) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)

				return
			}

			defer r.Body.Close()
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)

				return
			}

			var transForward proxy.TransparentForward
			if err := json.Unmarshal(body, &transForward); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)

				return
			}

			client := &http.Client{}
			var targetURL = ""
			if m.config.AppNoAgent {
				targetURL = fmt.Sprintf("%s:%d/%s", m.config.AppUrl, transForward.EdgePath.Port, transForward.EdgePath.InterfaceURL)
			} else {
				err, proxyPath := m.appAgent.GetProxyPath()
				if err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)

					return
				}
				targetURL = fmt.Sprintf("%s:%d/%s/%s", m.config.AppUrl, m.config.AppPort, proxyPath, transForward.EdgePath.InterfaceURL)
			}
			m.logger.Debug(proxy.TransparentForwardUrl, "targetURL", targetURL)

			req, err := http.NewRequest(r.Method, targetURL, bytes.NewReader([]byte(transForward.Payload)))
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)

				return
			}

			for key, values := range r.Header {
				for _, value := range values {
					req.Header.Add(key, value)
					m.logger.Debug(proxy.TransparentForwardUrl, key, value)
				}
			}

			resp, err := client.Do(req)
			if err != nil {
				http.Error(w, "Failed to connect to target server", http.StatusInternalServerError)

				return
			}
			defer resp.Body.Close()

			for key, value := range resp.Header {
				w.Header().Set(key, value[0])
			}
			w.WriteHeader(resp.StatusCode)
			if resp.Header.Get("Content-Type") == "text/event-stream" {
				reader := bufio.NewReader(resp.Body)
				for {
					line, err := reader.ReadBytes('\n')
					if err != nil {
						if err == io.EOF {

							return
						}
						m.logger.Warn(proxy.TransparentForwardUrl, "err", fmt.Sprintf("Error reading SSE stream: %v\n", err))

						return
					}

					_, err = w.Write(line)
					if err != nil {
						m.logger.Warn(proxy.TransparentForwardUrl, "err", fmt.Sprintf("Error writing to client: %v\n", err))

						return
					}

					w.(http.Flusher).Flush()
				}
			} else {
				io.Copy(w, resp.Body)
			}
		})

		if m.runningMode == RunningModeEdge {
			// keep edge peer alive
			err := m.relayClient.StartAlive(endpoint.SubscribeEvents())
			if err != nil {
				return nil, err
			}

			// do agent binding
			if !m.config.AppNoAgent {
				go func() {
					ticker := time.NewTicker(DefaultAppBindSyncDuration)
					for {
						<-ticker.C

						err = m.doAppNodeBind(endpointHost.ID().String())
						if err != nil {
							m.logger.Error("doAppNodeBind", "err", err.Error())
						}

						err, appOrigin := m.getAppOrigin()
						if err != nil {
							m.logger.Error("getAppOrigin", "err", err.Error())
						}
						endpoint.SetAppOrigin(appOrigin)

						m.logger.Info("binding", "NodeID", endpointHost.ID().String(), "AppOrigin", appOrigin)
					}
					ticker.Stop()
				}()
			}
		}

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
			err = syncer.Start(true)
			if err != nil {
				return nil, err
			}
			m.appPeerSyncer = syncer

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
				relayListenAddr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/tcp/%d", config.RelayAddr.IP.String(), config.RelayAddr.Port))
				if err != nil {
					return nil, err
				}
				relayServer, err := relay.NewRelayServer(logger, m.secretsManager, relayListenAddr, config.TransparentProxy.ProxyAddr, syncAppclient, config.RelayDiscovery, m.config.GenesisConfig.Bootnodes)
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

type jsonRPCHub struct {
	//*telepool.TelegramPool
	*network.Server
	application.SyncAppPeerClient
}

func (j jsonRPCHub) AddTele(tx *types.Telegram) (string, error) {
	//TODO implement AddTele
	panic("implement me")
}

func (j *jsonRPCHub) GetPeers() int {
	return len(j.Server.Peers())
}

// setupJSONRCP sets up the JSONRPC server, using the set configuration
func (s *Server) setupJSONRPC() error {
	hub := &jsonRPCHub{
		//TelegramPool:       s.telepool,
		Server:            s.edgeNetwork,
		SyncAppPeerClient: s.syncAppPeerClient,
	}
	conf := &web3.Config{
		Store:                    hub,
		Addr:                     s.config.JSONRPC.JSONRPCAddr,
		NetworkID:                uint64(s.config.GenesisConfig.NetworkId),
		ChainName:                s.config.GenesisConfig.Name,
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
		ChainName:                s.config.GenesisConfig.Name,
		AccessControlAllowOrigin: s.config.TransparentProxy.AccessControlAllowOrigin,
	}

	srv, err := proxy.NewTransportProxy(s.logger, conf, nil)
	if err != nil {
		return err
	}

	s.edgeProxyServer = srv

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
	if s.edgeNetwork != nil {
		if err := s.edgeNetwork.Close(); err != nil {
			s.logger.Error("failed to close networking", "err", err.Error())
		}
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
