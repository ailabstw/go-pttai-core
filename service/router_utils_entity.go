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
	"reflect"
	"sync"

	"github.com/ailabstw/go-pttai-core/common/types"
	"github.com/ailabstw/go-pttai-core/log"
	"github.com/ailabstw/go-pttai-core/p2p/discover"
	"github.com/ethereum/go-ethereum/common"
)

func (r *BaseRouter) getEntityFromHash(hash *common.Address, lock *sync.RWMutex, hashMap map[common.Address]*types.PttID) (Entity, error) {
	lock.RLock()
	defer lock.RUnlock()

	hashVal := *hash
	entityID, ok := hashMap[hashVal]
	if !ok {
		return nil, ErrInvalidData
	}
	idVal := *entityID
	entity, ok := r.entities[idVal]
	if !ok {
		return nil, ErrInvalidData
	}

	return entity, nil
}

func (r *BaseRouter) RegisterEntityPeerWithOtherUserID(e Entity, id *types.PttID, peerType PeerType, isLocked bool) error {

	if !isLocked {
		r.peerLock.RLock()
		defer r.peerLock.RUnlock()
	}

	myID := r.GetMyEntity().GetID()
	if reflect.DeepEqual(myID, id) {
		return nil
	}

	log.Debug("RegisterEntityPeerWithOtherUserID: to GetPeerByUserID", "e", e.GetID(), "id", id, "peerType", peerType)

	peer, err := r.GetPeerByUserID(id, true)

	log.Debug("RegisterEntityPeerWithOtherUserID: after GetPeerByUserID", "e", e.GetID(), "id", id, "peer", peer, "e", err)

	if err != nil {
		return err
	}
	if peer == nil {
		return nil
	}

	return e.PM().RegisterPeer(peer, peerType, false)
}

func (r *BaseRouter) RegisterEntity(e Entity, isLocked bool, isPeerLocked bool) error {
	if !isLocked {
		r.entityLock.Lock()
		defer r.entityLock.Unlock()
	}

	id := e.GetID()
	r.entities[*id] = e

	log.Debug("RegisterEntity: to registerEntityPeers")

	return r.registerEntityPeers(e, isPeerLocked)
}

func (r *BaseRouter) registerEntityPeers(e Entity, isLocked bool) error {
	if !isLocked {
		r.peerLock.RLock()
		defer r.peerLock.RUnlock()
	}

	log.Debug("registerEntityPeers: after lock")

	toMyPeers := make([]*PttPeer, 0)
	toImportantPeers := make([]*PttPeer, 0)
	toMemberPeers := make([]*PttPeer, 0)
	toPendingPeers := make([]*PttPeer, 0)

	pm := e.PM()

	// my-peers: always my-peer and register the entity
	var peer *PttPeer
	log.Debug("registerEntityPeers: to myPeers")
	for _, peer = range r.myPeers {
		pm.RegisterPeer(peer, PeerTypeMe, false)
	}

	// hub-peers
	log.Debug("registerEntityPeers: to hubPeers")
	for _, peer = range r.hubPeers {
		if pm.IsMyDevice(peer) {
			pm.RegisterPeer(peer, PeerTypeMe, false)
		} else if pm.IsImportantPeer(peer) {
			pm.RegisterPeer(peer, PeerTypeImportant, false)
		} else if pm.IsMemberPeer(peer) {
			pm.RegisterPeer(peer, PeerTypeMember, false)
		} else if pm.IsPendingPeer(peer) {
			pm.RegisterPendingPeer(peer, false)
		}
	}

	// important-peers
	toRemovePeers := make([]*discover.NodeID, 0)
	log.Debug("registerEntityPeers: to importantPeers")
	for _, peer = range r.importantPeers {
		if pm.IsMyDevice(peer) {
			log.Debug("registerEntityPeers: important-to-me", "peer", peer)
			pm.RegisterPeer(peer, PeerTypeMe, false)
			toMyPeers = append(toMyPeers, peer)
			toRemovePeers = append(toRemovePeers, peer.GetID())
		} else if pm.IsImportantPeer(peer) {
			pm.RegisterPeer(peer, PeerTypeImportant, false)
		} else if pm.IsMemberPeer(peer) {
			pm.RegisterPeer(peer, PeerTypeMember, false)
		} else if pm.IsPendingPeer(peer) {
			pm.RegisterPendingPeer(peer, false)
		}
	}
	for _, nodeID := range toRemovePeers {
		delete(r.importantPeers, *nodeID)
	}

	// member-peers
	toRemovePeers = make([]*discover.NodeID, 0)
	log.Debug("registerEntityPeers: to memberPeers")
	for _, peer = range r.memberPeers {
		if pm.IsMyDevice(peer) {
			log.Debug("registerEntityPeers: member-to-me", "peer", peer)
			pm.RegisterPeer(peer, PeerTypeMe, false)
			toMyPeers = append(toMyPeers, peer)
			toRemovePeers = append(toRemovePeers, peer.GetID())
		} else if pm.IsImportantPeer(peer) {
			pm.RegisterPeer(peer, PeerTypeImportant, false)
			toImportantPeers = append(toImportantPeers, peer)
			toRemovePeers = append(toRemovePeers, peer.GetID())
		} else if pm.IsMemberPeer(peer) {
			pm.RegisterPeer(peer, PeerTypeMember, false)
		} else if pm.IsPendingPeer(peer) {
			pm.RegisterPendingPeer(peer, false)
		}
	}
	for _, nodeID := range toRemovePeers {
		delete(r.memberPeers, *nodeID)
	}

	// pending-peers
	toRemovePeers = make([]*discover.NodeID, 0)
	log.Debug("registerEntityPeers: to pendingPeers", "pendingPeers", len(r.pendingPeers))
	for _, peer = range r.pendingPeers {
		if pm.IsMyDevice(peer) {
			pm.RegisterPeer(peer, PeerTypeMe, false)
			log.Debug("registerEntityPeers: pending-to-me", "peer", peer)
			toMyPeers = append(toMyPeers, peer)
			toRemovePeers = append(toRemovePeers, peer.GetID())
		} else if pm.IsImportantPeer(peer) {
			pm.RegisterPeer(peer, PeerTypeImportant, false)
			toImportantPeers = append(toImportantPeers, peer)
			toRemovePeers = append(toRemovePeers, peer.GetID())
		} else if pm.IsMemberPeer(peer) {
			pm.RegisterPeer(peer, PeerTypeMember, false)
			toMemberPeers = append(toMemberPeers, peer)
			toRemovePeers = append(toRemovePeers, peer.GetID())
		} else if pm.IsPendingPeer(peer) {
			pm.RegisterPendingPeer(peer, false)
		}
	}
	for _, nodeID := range toRemovePeers {
		delete(r.memberPeers, *nodeID)
	}

	// random-peers
	toRemovePeers = make([]*discover.NodeID, 0)
	log.Debug("registerEntityPeers: to randomPeers", "randomPeers", len(r.randomPeers))
	for _, peer = range r.randomPeers {
		if pm.IsMyDevice(peer) {
			log.Debug("registerEntityPeers: random-to-me", "peer", peer)
			pm.RegisterPeer(peer, PeerTypeMe, false)
			toMyPeers = append(toMyPeers, peer)
			toRemovePeers = append(toRemovePeers, peer.GetID())
		} else if pm.IsImportantPeer(peer) {
			pm.RegisterPeer(peer, PeerTypeImportant, false)
			toImportantPeers = append(toImportantPeers, peer)
			toRemovePeers = append(toRemovePeers, peer.GetID())
		} else if pm.IsMemberPeer(peer) {
			pm.RegisterPeer(peer, PeerTypeMember, false)
			toMemberPeers = append(toMemberPeers, peer)
			toRemovePeers = append(toRemovePeers, peer.GetID())
		} else if pm.IsPendingPeer(peer) {
			pm.RegisterPeer(peer, PeerTypePending, false)
			toPendingPeers = append(toPendingPeers, peer)
			toRemovePeers = append(toRemovePeers, peer.GetID())
		}
	}
	for _, nodeID := range toRemovePeers {
		delete(r.randomPeers, *nodeID)
	}

	// to my-peers
	log.Debug("registerEntityPeers", "toMyPeers", len(toMyPeers))
	for _, peer = range toMyPeers {
		//id := peer.ID()
		r.SetPeerType(peer, PeerTypeMe, false, true)
		// p.myPeers[id] = peer
	}

	// to important-peers
	log.Debug("registerEntityPeers", "toImportantPeers", len(toImportantPeers))
	for _, peer = range toImportantPeers {
		//id := peer.ID()
		r.SetPeerType(peer, PeerTypeImportant, false, true)
		// p.importantPeers[id] = peer
	}

	// to member
	log.Debug("registerEntityPeers", "toMemberPeers", len(toMemberPeers))
	for _, peer = range toMemberPeers {
		//id := peer.ID()
		r.SetPeerType(peer, PeerTypeMember, false, true)
		//p.memberPeers[id] = peer
	}

	// to pending
	log.Debug("registerEntityPeers", "toPendingPeers", len(toPendingPeers))
	for _, peer = range toPendingPeers {
		//id := peer.ID()
		r.SetPeerType(peer, PeerTypePending, false, true)
		// p.pendingPeers[id] = peer
	}

	log.Debug("registerEntityPeers: done")

	return nil
}

func (r *BaseRouter) UnregisterEntity(e Entity, isLocked bool) error {
	if !isLocked {
		r.entityLock.Lock()
		defer r.entityLock.Unlock()
	}

	id := e.GetID()
	delete(r.entities, *id)

	r.peerLock.Lock()
	defer r.peerLock.Unlock()

	return nil
}

func (r *BaseRouter) GetEntities() map[types.PttID]Entity {
	return r.entities
}
