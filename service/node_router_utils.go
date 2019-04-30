// Copyright 2019 The go-pttai Authors
// This file is part of the go-pttai library.
//
// The go-pttai library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-pttai library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-pttai library. If not, see <http://www.gnu.org/licenses/>.

package service

import (
	"github.com/ailabstw/go-pttai-core/common/types"
	"github.com/ailabstw/go-pttai-core/p2p"
	"github.com/ailabstw/go-pttai-core/p2p/discover"
	"github.com/ailabstw/go-pttai-core/rpc"
)

func (r *BaseRouter) generateProtocols() []p2p.Protocol {
	subProtocols := make([]p2p.Protocol, 0, len(ProtocolVersions))

	for i, version := range ProtocolVersions {
		protocol := p2p.Protocol{
			Name:     ProtocolName,
			Version:  version,
			Length:   ProtocolLengths[i],
			Run:      r.generateRun(version),
			NodeInfo: r.generateNodeInfo(),
			PeerInfo: r.generatePeerInfo(),
		}

		subProtocols = append(subProtocols, protocol)
	}

	return subProtocols
}

/*
generateRun generates run in Protocol (PttService)
(No need to do sync in the ptt-layer for now, because there is no information needs to sync in the ptt-layer.)

    1. set up ptt-peer.
    2. peerWG.
    3. handle-peer.
*/
func (r *BaseRouter) generateRun(version uint) func(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	return func(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
		// 1. pttPeer
		pttPeer, err := r.NewPeer(version, peer, rw)
		if err != nil {
			return err
		}

		// 2. peerWG
		r.peerWG.Add(1)
		defer r.peerWG.Done()

		// 3. handle peer
		err = r.HandlePeer(pttPeer)

		return err
	}
}

func (r *BaseRouter) generateNodeInfo() func() interface{} {
	return func() interface{} {
		return r.nodeInfo()
	}
}

func (r *BaseRouter) generatePeerInfo() func(id discover.NodeID) interface{} {
	return func(id discover.NodeID) interface{} {
		r.peerLock.RLock()
		defer r.peerLock.RUnlock()

		peer := r.GetPeer(&id, true)
		if peer == nil {
			return nil
		}

		return peer.Info()
	}
}

func (r *BaseRouter) routerAPIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "ptt",
			Version:   "1.0",
			Service:   NewPrivateAPI(r),

			Public: IsPrivateAsPublic,
		},
	}
}

func (r *BaseRouter) nodeInfo() interface{} {
	peers := len(r.myPeers) + len(r.importantPeers) + len(r.memberPeers) + len(r.randomPeers)
	var userID *types.PttID
	if r.myEntity != nil {
		userID = r.myEntity.GetID()
	}

	return &NodeRouterInfo{
		NodeID:   r.myNodeID,
		UserID:   userID,
		Peers:    peers,
		Entities: len(r.entities),
		Services: len(r.services),
	}
}
