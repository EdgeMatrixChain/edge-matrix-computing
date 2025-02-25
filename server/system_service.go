package server

import (
	"context"
	"fmt"
	"github.com/emc-protocol/edge-matrix-computing/server/proto"
	"github.com/emc-protocol/edge-matrix-core/core/network/common"
	"github.com/libp2p/go-libp2p/core/peer"
	empty "google.golang.org/protobuf/types/known/emptypb"
)

type systemService struct {
	proto.UnimplementedSystemServer

	server *Server
}

// GetStatus returns the current system status, in the form of:
//
// P2PAddr: <libp2pAddress>
func (s *systemService) GetStatus(ctx context.Context, req *empty.Empty) (*proto.ServerStatus, error) {

	status := &proto.ServerStatus{
		P2PAddr: common.AddrInfoToString(s.server.edgeNetwork.AddrInfo()),
	}

	return status, nil
}

// PeersAdd implements the 'peers add' operator service
func (s *systemService) PeersAdd(_ context.Context, req *proto.PeersAddRequest) (*proto.PeersAddResponse, error) {
	if joinErr := s.server.JoinPeer(req.Id); joinErr != nil {
		return &proto.PeersAddResponse{
			Message: "Unable to successfully add peer",
		}, joinErr
	}

	return &proto.PeersAddResponse{
		Message: "Peer address marked ready for dialing",
	}, nil
}

// PeersStatus implements the 'peers status' operator service
func (s *systemService) PeersStatus(ctx context.Context, req *proto.PeersStatusRequest) (*proto.Peer, error) {
	peerID, err := peer.Decode(req.Id)
	if err != nil {
		return nil, err
	}

	protoPeer, err := s.getPeer(peerID)
	if err != nil {
		return nil, err
	}

	return protoPeer, nil
}

// getPeer returns a specific proto.Peer using the peer ID
func (s *systemService) getPeer(id peer.ID) (*proto.Peer, error) {
	protocols, err := s.server.edgeNetwork.GetProtocols(id)
	if err != nil {
		return nil, err
	}

	info := s.server.edgeNetwork.GetPeerInfo(id)

	addrs := []string{}
	for _, addr := range info.Addrs {
		addrs = append(addrs, addr.String())
	}
	protoList := make([]string, 0)
	for prot := range protocols {
		protoList = append(protoList, string(prot))
	}

	protoPeer := &proto.Peer{
		Id:        id.String(),
		Protocols: protoList,
		Addrs:     addrs,
	}

	return protoPeer, nil
}

// PeersList implements the 'peers list' operator service
func (s *systemService) PeersList(
	ctx context.Context,
	req *empty.Empty,
) (*proto.PeersListResponse, error) {
	resp := &proto.PeersListResponse{
		Peers: []*proto.Peer{},
	}

	if s.server.edgeNetwork != nil {
		edgePeers := s.server.edgeNetwork.Peers()
		for _, p := range edgePeers {
			peer, err := s.getPeer(p.Info.ID)
			if err != nil {
				return nil, err
			}

			resp.Peers = append(resp.Peers, peer)
		}
	}

	return resp, nil
}

// RelayConnections implements the 'peers relay' operator service
func (s *systemService) RelayConnections(
	ctx context.Context,
	req *empty.Empty,
) (*proto.RelayConnectionsCount, error) {
	resp := &proto.RelayConnectionsCount{
		Connected:       0,
		MaxReservations: 0,
	}

	if s.server.relayServer != nil {
		relayHost := s.server.relayServer.GetHost()
		resp.Connected = uint64(len(relayHost.Network().Peers()))
		resp.MaxReservations = int64(s.server.relayServer.MaxReservations)
	}

	return resp, nil
}

// PeersRelayList implements the 'peers relaylist' operator service
func (s *systemService) PeersRelayList(
	ctx context.Context,
	req *empty.Empty,
) (*proto.PeersListResponse, error) {
	if s.server.relayClient == nil {
		return nil, nil
	}

	resp := &proto.PeersListResponse{
		Peers: []*proto.Peer{},
	}

	peers := s.server.relayClient.GetBootnodes()
	for _, p := range peers {
		addrs := []string{}
		for _, addr := range p.Addrs {
			addrs = append(addrs, addr.String())
		}

		resp.Peers = append(resp.Peers, &proto.Peer{
			Id:    p.ID.String(),
			Addrs: addrs,
		})
	}

	return resp, nil
}

// RelayStatus implements the 'peers relay' operator service
func (s *systemService) RelayStatus(
	ctx context.Context,
	req *empty.Empty,
) (*proto.Peer, error) {

	if s.server.relayClient == nil {
		return nil, nil
	}
	relayPeers := s.server.relayClient.RelayPeers()
	if relayPeers != nil && len(relayPeers) > 0 {
		addrs := []string{}
		for _, addr := range relayPeers[0].Info.Info.Addrs {
			addrs = append(addrs, addr.String())
		}
		resv := relayPeers[0].Reservation
		protoPeer := &proto.Peer{
			Id:          relayPeers[0].Info.Info.ID.String(),
			Addrs:       addrs,
			Reservation: fmt.Sprintf("LimitData:%d, LimitDuration:%v, Expiration:%v, Addrs:%v", resv.LimitData, resv.LimitDuration, resv.Expiration, resv.Addrs),
		}
		return protoPeer, nil
	}

	return &proto.Peer{
		Id:    "",
		Addrs: make([]string, 0),
	}, nil
}
