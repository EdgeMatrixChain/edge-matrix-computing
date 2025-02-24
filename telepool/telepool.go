package telepool

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/emc-protocol/edge-matrix-core/core/application"
	"github.com/emc-protocol/edge-matrix-core/core/application/proof"
	"github.com/emc-protocol/edge-matrix-core/core/network"
	"github.com/emc-protocol/edge-matrix-core/core/types"
	"github.com/hashicorp/go-hclog"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/umbracle/fastrlp"
	"time"
)

// indicates origin of a transaction
type teleOrigin int

const (
	local  teleOrigin = iota // json-RPC/gRPC endpoints
	gossip                   // gossip protocol
)

// errors
var (
	ErrIntrinsicGas            = errors.New("intrinsic gas too low")
	ErrBlockLimitExceeded      = errors.New("exceeds block gas limit")
	ErrNegativeValue           = errors.New("negative value")
	ErrExtractSignature        = errors.New("cannot extract signature")
	ErrInvalidSender           = errors.New("invalid sender")
	ErrInvalidProvider         = errors.New("invalid provider")
	ErrTxPoolOverflow          = errors.New("txpool is full")
	ErrUnderpriced             = errors.New("transaction underpriced")
	ErrNonceTooLow             = errors.New("nonce too low")
	ErrNonceTooHigh            = errors.New("nonce too high")
	ErrInsufficientFunds       = errors.New("insufficient funds for gas * price + value")
	ErrInvalidAccountState     = errors.New("invalid account state")
	ErrAlreadyKnown            = errors.New("already known")
	ErrOversizedData           = errors.New("oversized data")
	ErrMaxEnqueuedLimitReached = errors.New("maximum number of enqueued transactions reached")
	ErrRejectFutureTx          = errors.New("rejected future tx due to low slots")
	ErrSmartContractRestricted = errors.New("smart contract deployment restricted")
)

// EdgeCallPrecompile is and address of edge call precompile
var EdgeCallPrecompile = types.StringToAddress("0x3001")

func (o teleOrigin) String() (s string) {
	switch o {
	case local:
		s = "local"
	case gossip:
		s = "gossip"
	}

	return
}

const (
	txSlotSize  = 32 * 1024
	txMaxSize   = 128 * 1024 // 128
	topicNameV1 = "tele/0.3"

	// maximum allowed number of times an account
	// was excluded from block building (ibft.writeTransactions)
	maxAccountDemotions uint64 = 10

	// maximum allowed number of consecutive blocks that don't have the account's transaction
	maxAccountSkips = uint64(10)
	pruningCooldown = 5000 * time.Millisecond

	// txPoolMetrics is a prefix used for txpool-related metrics
	txPoolMetrics = "telepool"
)

var marshalArenaPool fastrlp.ArenaPool

type enqueueRequest struct {
	tele *types.Telegram
}

type signer interface {
	Sender(tele *types.Telegram) (types.Address, error)
	Provider(tele *types.Telegram) (types.Address, error)
}

type providerSigner interface {
	Provider(tele *types.Telegram) (types.Address, error)
}

// A promoteRequest is created each time some account
// is eligible for promotion. This request is signaled
// on 2 occasions:
//
// 1. When an enqueued transaction's nonce is
// not greater than the expected (account's nextNonce).
// == nextNonce - transaction is expected (addTele)
// < nextNonce - transaction was demoted (Demote)
//
// 2. When an account's nextNonce is updated (during ResetWithHeader)
// and the first enqueued transaction matches the new nonce.
type promoteRequest struct {
	account types.Address
}

type Config struct {
	MaxSlots           uint64
	MaxAccountEnqueued uint64
}

type TelepoolStore interface {
	GetRelayHost() host.Host
	GetNetworkHost() host.Host
	GetAppPeer(id string) *application.AppPeer
}

type TelegramPool struct {
	logger         hclog.Logger
	signer         signer
	providerSigner providerSigner

	// networking stack
	topic *network.Topic

	// Syncer interface
	//appSyncer application.Syncer
	store TelepoolStore
	// gauge for measuring pool capacity
	gauge slotGauge

	// channels on which the pool's event loop
	// does dispatching/handling requests.
	enqueueReqCh chan enqueueRequest
	promoteReqCh chan promoteRequest
	pruneCh      chan struct{}

	// shutdown channel
	shutdownCh chan struct{}

	// Event manager for telepool events
	//eventManager *eventManager

	// indicates which txpool operator commands should be implemented
	//proto.UnimplementedTxnPoolOperatorServer

	// pending is the list of pending and ready transactions. This variable
	// is accessed with atomics
	pending int64
}

// NewTelegramPool returns a new pool for processing incoming telegram.
func NewTelegramPool(
	logger hclog.Logger,
	config *Config,
	store TelepoolStore,
	signer signer,
) *TelegramPool {
	pool := &TelegramPool{
		logger: logger.Named("telepool"),
		gauge:  slotGauge{height: 0, max: config.MaxSlots},
		store:  store,
		signer: signer,
		//	main loop channels
		enqueueReqCh: make(chan enqueueRequest),
		promoteReqCh: make(chan promoteRequest),
		pruneCh:      make(chan struct{}),
		shutdownCh:   make(chan struct{}),
	}

	return pool
}

// AddTele adds a new telegram to the pool (sent from json-RPC/gRPC endpoints)
// and broadcasts it to the network (if enabled).
func (p *TelegramPool) AddTele(tele *types.Telegram) (string, error) {
	resp := &proof.EdgeResponse{}
	if tele.To != nil && *tele.To == EdgeCallPrecompile {
		input := tele.Input
		call := &application.EdgeCall{}
		if err := json.Unmarshal(input, &call); err != nil {
			return "", err
		}
		host := p.store.GetRelayHost()

		relayAddr, addr := p.getAppPeerAddr(call.PeerId)
		p.logger.Debug("edge call", "PeerId", call.PeerId, "Endpoint", call.Endpoint, "addr", addr, "Relay", relayAddr)
		if relayAddr != "" || addr != "" {
			err := p.addAddrToHost(call.PeerId, host, addr, relayAddr)
			if err != nil {
				return "", err
			}
		}

		respBuf, callErr := application.Call(host, application.ProtoTagEcApp, call)
		if callErr != nil {
			return "", callErr
		}

		err := resp.UnmarshalRLP(respBuf)
		if err != nil {
			return "", err
		}
		tele.RespFrom = resp.From
		tele.RespR = resp.R
		tele.RespV = resp.V
		tele.RespS = resp.S
		tele.RespHash = resp.Hash
		if len(resp.RespString) > 0 {
			return resp.RespString, nil
		} else {
			return "", nil
		}
	}

	return "", nil
}

func (p *TelegramPool) addAddrToHost(peerId string, host host.Host, addr string, relayAddr string) error {
	if relayAddr != "" {
		targetRelayInfo, err := peer.AddrInfoFromString(fmt.Sprintf("%s/p2p-circuit/p2p/%s", relayAddr, peerId))
		if err != nil {
			return err
		}
		host.Peerstore().AddAddrs(targetRelayInfo.ID, targetRelayInfo.Addrs, peerstore.AddressTTL)
	} else if addr != "" {
		addrInfo, err := peer.AddrInfoFromString(fmt.Sprintf("%s/p2p/%s", addr, peerId))
		if err != nil {
			return err
		}
		host.Peerstore().AddAddrs(addrInfo.ID, addrInfo.Addrs, peerstore.RecentlyConnectedAddrTTL)
	}
	return nil
}

//
//func (p *TelegramPool) newTempHost() (host.Host, error) {
//	var r io.Reader
//	r = rand.Reader
//	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
//	if err != nil {
//		return nil, err
//	}
//	listen, _ := ma.NewMultiaddr("/ip4/127.0.0.1/tcp/10001")
//	clientHost, err := libp2p.New(
//		libp2p.ListenAddrs(listen),
//		libp2p.Security(noise.ID, noise.New),
//		libp2p.Identity(prvKey),
//	)
//	if err != nil {
//		return nil, err
//	}
//	return clientHost, nil
//}

func (p *TelegramPool) getAppPeerAddr(peerId string) (relayAddr string, addr string) {
	appPeer := p.store.GetAppPeer(peerId)
	if appPeer != nil {
		relayAddr = appPeer.Relay
		addr = appPeer.Addr
		return
	}
	return "", ""
}

// validateTele ensures the telegram conforms to specific
// constraints before entering the pool.
func (p *TelegramPool) validateTele(tele *types.Telegram) error {
	// Check the transaction size to overcome DOS Attacks
	if uint64(len(tele.MarshalRLP())) > txMaxSize {
		return ErrOversizedData
	}

	// Check if the transaction is signed properly

	// Extract the sender
	from, signerErr := p.signer.Sender(tele)
	if signerErr != nil {
		return ErrExtractSignature
	}

	p.logger.Debug(fmt.Sprintf("validateTele from: %s", from.String()))

	// Extract the provider
	if tele.RespFrom != types.ZeroAddress {
		respFrom, signerErr := p.signer.Provider(tele)
		if signerErr != nil {
			return ErrExtractSignature
		}
		p.logger.Debug(fmt.Sprintf("validateTele RespFrom:%s, provider: %s", tele.RespFrom, respFrom.String()))
		if respFrom != tele.RespFrom {
			return ErrInvalidProvider

		}
	}
	// testAddress
	//from := types.StringToAddress("0x68b95f67a32935e3ed85600F558b74E0d2747120")

	// If the from field is set, check that
	// it matches the signer
	if tele.From != types.ZeroAddress &&
		tele.From != from {
		return ErrInvalidSender
	}

	// If no address was set, update it
	if tele.From == types.ZeroAddress {
		tele.From = from
	}

	return nil
}

// Close shuts down the pool's main loop.
func (p *TelegramPool) Close() {
	p.shutdownCh <- struct{}{}
}

// SetSigner sets the signer the pool will use
// to validate a telegram's signature.
func (p *TelegramPool) SetSigner(s signer) {
	p.signer = s
}
