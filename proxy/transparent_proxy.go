package proxy

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/emc-protocol/edge-matrix-core/core/application"
	p2phttp "github.com/libp2p/go-libp2p-http"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/emc-protocol/edge-matrix-core/core/versioning"
	"github.com/hashicorp/go-hclog"
)

const TransparentForwardUrl = "/transparent_forward"

type serverType int

const (
	serverIPC serverType = iota
	serverHTTP
	serverWS
)

func (s serverType) String() string {
	switch s {
	case serverIPC:
		return "ipc"
	case serverHTTP:
		return "http"
	case serverWS:
		return "ws"
	default:
		panic("BUG: Not expected")
	}
}

// TransparentProxy is an API consensus
type TransparentProxy struct {
	logger hclog.Logger
	config *Config
}

// TransparentProxyStore defines all the methods required
// by all the proxy endpoints
type TransparentProxyStore interface {
	GetRelayHost() host.Host
	GetNetworkHost() host.Host
	GetAppPeer(id string) *application.AppPeer
	ValidateBearer(bearer string) bool
}

type Config struct {
	Store                    TransparentProxyStore
	Addr                     *net.TCPAddr
	NetworkID                uint64
	ChainName                string
	AccessControlAllowOrigin []string
}

// NewTransportProxy returns the TransparentProxy http server
func NewTransportProxy(logger hclog.Logger, config *Config, middlewareFactory MiddlewareFactory) (*TransparentProxy, error) {
	srv := &TransparentProxy{
		logger: logger.Named("transport-proxy"),
		config: config,
	}

	// start http server
	if err := srv.setupHTTP(middlewareFactory); err != nil {
		return nil, err
	}

	return srv, nil
}

type MiddlewareFactory func(config *Config) func(http.Handler) http.Handler

func (j *TransparentProxy) setupHTTP(middlewareFactory MiddlewareFactory) error {
	j.logger.Info("http server started", "addr", j.config.Addr.String())

	lis, err := net.Listen("tcp", j.config.Addr.String())
	if err != nil {
		return err
	}

	// NewServeMux must be used, as it disables all debug features.
	// For some strange reason, with DefaultServeMux debug/vars is always enabled (but not debug/pprof).
	// If pprof need to be enabled, this should be DefaultServeMux
	mux := http.NewServeMux()

	// The middleware factory returns a handler, so we need to wrap the handler function properly.
	proxyHandler := http.HandlerFunc(j.handle)

	if middlewareFactory != nil {
		mux.Handle("/", middlewareFactory(j.config)(proxyHandler))
	} else {
		mux.Handle("/", j.defaultMiddlewareFactory()(proxyHandler))
	}

	// TODO implement websocket handler
	//mux.HandleFunc("/edge_ws", j.handleWs)

	srv := http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 60 * time.Second,
	}

	go func() {
		if err := srv.Serve(lis); err != nil {
			j.logger.Error("closed http connection", "err", err)
		}
	}()

	return nil
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

// The BearerMiddlewareFactory builds a middleware which enables authorization with Bearer.
func (j *TransparentProxy) BearerMiddlewareFactory() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// verify bearer
			bearer := getBearer(r)
			if bearer == "" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)

				return
			}
			if !j.config.Store.ValidateBearer(bearer) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)

				return
			}

			origin := r.Header.Get("Origin")
			for _, allowedOrigin := range j.config.AccessControlAllowOrigin {
				if allowedOrigin == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")

					break
				}

				if allowedOrigin == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)

					break
				}
			}

			// add Header: Forwarded(RFC 7239),
			// e.g Forwarded: proto=http; host="example.com:8080"; for="client_ip"
			r.Header.Add("Forwarded", fmt.Sprintf("proto=%s; host=%s; for=%s", "p2phttp", r.Host, r.RemoteAddr))

			pathInfo, err := ParseEdgePath(r)
			if err != nil {
				_, _ = w.Write([]byte(err.Error()))
				return
			}
			j.logger.Info("handle", "NodeID", pathInfo.NodeID, "Port", pathInfo.Port, "InterfaceURL", pathInfo.InterfaceURL)

			// add Header: X-Forwarded-Port, X-Forwarded-Host
			r.Header.Add("X-Forwarded-Port", strconv.Itoa(pathInfo.Port))
			r.Header.Add("X-Forwarded-Host", j.config.Store.GetRelayHost().ID().String())

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "EdgePath", pathInfo)))
		})
	}
}

// The defaultMiddlewareFactory builds a middleware which enables CORS using the provided config.
func (j *TransparentProxy) defaultMiddlewareFactory() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			for _, allowedOrigin := range j.config.AccessControlAllowOrigin {
				if allowedOrigin == "*" {
					w.Header().Set("Access-Control-Allow-Origin", "*")

					break
				}

				if allowedOrigin == origin {
					w.Header().Set("Access-Control-Allow-Origin", origin)

					break
				}
			}
			// add Header: Forwarded(RFC 7239),
			// e.g Forwarded: proto=http; host="example.com:8080"; for="client_ip"
			r.Header.Add("Forwarded", fmt.Sprintf("proto=%s; host=%s; for=%s", "p2phttp", r.Host, r.RemoteAddr))

			pathInfo, err := ParseEdgePath(r)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			// add Header: X-Forwarded-Port, X-Forwarded-Host
			r.Header.Add("X-Forwarded-Port", strconv.Itoa(pathInfo.Port))
			r.Header.Add("X-Forwarded-Host", j.config.Store.GetRelayHost().ID().String())

			next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), "EdgePath", pathInfo)))
		})
	}
}

func (j *TransparentProxy) handle(w http.ResponseWriter, req *http.Request) {
	switch req.Method {
	case "POST":
		j.handlePostRequest(w, req)
	case "GET":
		j.handleGetRequest(w)
	case "OPTIONS":
		// nothing to return
	default:
		_, _ = w.Write([]byte("method " + req.Method + " not allowed"))
	}
}

type EdgePath struct {
	NodeID       string `json:"node_id"`
	Port         int    `json:"port"`
	InterfaceURL string `json:"interface_url"`
}

type TransparentForward struct {
	EdgePath EdgePath `json:"edge_path"`
	Payload  string   `json:"payload"`
}

func ParseEdgePath(req *http.Request) (*EdgePath, error) {
	path := req.URL.Path
	parts := strings.Split(path, "/")

	if len(parts) < 4 {
		return nil, fmt.Errorf("invalid path format: expected at least 4 parts, got %d", len(parts))
	}

	nodeID := parts[2]
	port := parts[3]
	interfaceURL := strings.Join(parts[4:], "/")

	decodedNodeID, err := url.QueryUnescape(nodeID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nodeID: %w", err)
	}

	decodedPort, err := strconv.Atoi(port)
	if err != nil {
		return nil, fmt.Errorf("failed to decode port: %w", err)
	}

	decodedInterfaceURL, err := url.QueryUnescape(interfaceURL)
	if err != nil {
		return nil, fmt.Errorf("failed to decode interfaceURL: %w", err)
	}

	return &EdgePath{
		NodeID:       decodedNodeID,
		Port:         decodedPort,
		InterfaceURL: decodedInterfaceURL,
	}, nil
}

func (j *TransparentProxy) handlePostRequest(w http.ResponseWriter, req *http.Request) {
	pathInfo, ok := req.Context().Value("EdgePath").(*EdgePath)
	if !ok {
		http.Error(w, "Invalid edge path", http.StatusBadRequest)
		return
	}

	j.logger.Info("handle", "NodeID", pathInfo.NodeID, "Port", pathInfo.Port, "InterfaceURL", pathInfo.InterfaceURL)

	// TODO verify NodeID by whitelist

	defer req.Body.Close()
	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// log request
	j.logger.Debug("handle", "request", string(body))

	clientHost := j.config.Store.GetRelayHost()
	// query node in PeerStore
	appPeer := j.config.Store.GetAppPeer(pathInfo.NodeID)
	if appPeer == nil {
		http.Error(w, "Failed to find node", http.StatusServiceUnavailable)
		return
	}

	//targetRelayInfo, err := peer.AddrInfoFromString(fmt.Sprintf("%s/p2p/%s/p2p-circuit/p2p/%s", j.config.Store.GetRelayHost().Addrs()[0].String(), j.config.Store.GetRelayHost().ID().String(), pathInfo.NodeID))
	if appPeer.Relay != "" {
		targetRelayInfo, err := peer.AddrInfoFromString(fmt.Sprintf("%s/p2p-circuit/p2p/%s", appPeer.Relay, pathInfo.NodeID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		clientHost.Peerstore().AddAddrs(targetRelayInfo.ID, targetRelayInfo.Addrs, peerstore.RecentlyConnectedAddrTTL)
	} else if appPeer.Addr != "" {
		addrInfo, err := peer.AddrInfoFromString(fmt.Sprintf("%s/p2p/%s", appPeer.Addr, pathInfo.NodeID))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		clientHost.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, peerstore.RecentlyConnectedAddrTTL)
	} else {
		http.Error(w, "Failed to find addr of node", http.StatusServiceUnavailable)
		return
	}

	tr := &http.Transport{}
	tr.RegisterProtocol("libp2p", p2phttp.NewTransport(clientHost, p2phttp.ProtocolOption(application.ProtoTagEcApp)))
	client := &http.Client{Transport: tr}

	transparentForwardData := &TransparentForward{
		EdgePath: *pathInfo,
		Payload:  string(body),
	}
	data, err := json.Marshal(transparentForwardData)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	targetURL := fmt.Sprintf("libp2p://%s%s", pathInfo.NodeID, TransparentForwardUrl)
	request, err := http.NewRequest(req.Method, targetURL, bytes.NewBufferString(string(data)))
	if err != nil {
		http.Error(w, "Failed to create p2p request", http.StatusInternalServerError)
		return
	}
	// forward headers
	for key, values := range req.Header {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}

	// do request
	resp, err := client.Do(request)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
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
					j.logger.Debug("handlePostRequest", "msg", "SSE stream closed by server")
					return
				}
				j.logger.Warn("handlePostRequest", "err", fmt.Sprintf("Error reading SSE stream: %v\n", err))
				return
			}

			_, err = w.Write(line)
			if err != nil {
				j.logger.Warn("handlePostRequest", "err", fmt.Sprintf("Error writing to client: %v\n", err))
				return
			}

			w.(http.Flusher).Flush()
		}
	} else {
		io.Copy(w, resp.Body)
	}
}

type GetResponse struct {
	Name    string `json:"name"`
	ChainID uint64 `json:"chain_id"`
	Version string `json:"version"`
}

func (j *TransparentProxy) handleGetRequest(writer io.Writer) {
	data := &GetResponse{
		Name:    j.config.ChainName,
		ChainID: j.config.NetworkID,
		Version: versioning.Version,
	}

	resp, err := json.Marshal(data)
	if err != nil {
		_, _ = writer.Write([]byte(err.Error()))
	}

	if _, err = writer.Write(resp); err != nil {
		_, _ = writer.Write([]byte(err.Error()))
	}
}
