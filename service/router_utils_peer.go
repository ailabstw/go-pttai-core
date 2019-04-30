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
	mrand "math/rand"
	"reflect"

	"github.com/ailabstw/go-pttai-core/common/types"
	"github.com/ailabstw/go-pttai-core/log"
	"github.com/ailabstw/go-pttai-core/p2p"
	"github.com/ailabstw/go-pttai-core/p2p/discover"
	"github.com/ethereum/go-ethereum/common"
)

/**********
 * Peer
 **********/

/*
NewPeer inits PttPeer
*/
func (r *BaseRouter) NewPeer(version uint, peer *p2p.Peer, rw p2p.MsgReadWriter) (*PttPeer, error) {
	meteredMsgReadWriter, err := NewBaseMeteredMsgReadWriter(rw, version)
	if err != nil {
		return nil, err
	}
	return NewPttPeer(version, peer, meteredMsgReadWriter, r)
}

/*
HandlePeer handles peer
	1. Basic handshake
	2. AddNewPeer (defer RemovePeer)
	3. init read/write
	4. for-loop handle-message
*/
func (r *BaseRouter) HandlePeer(peer *PttPeer) error {
	log.Debug("HandlePeer: start", "peer", peer)
	defer log.Debug("HandlePeer: done", "peer", peer)

	// 1. basic handshake
	err := peer.Handshake(r.networkID)
	if err != nil {
		return err
	}

	// 2. add new peer (defer remove-peer)
	err = r.AddNewPeer(peer)
	if err != nil {
		return err
	}
	defer r.RemovePeer(peer, false)

	// 3. init read-write
	r.RWInit(peer, peer.Version())

	// 4. for-loop handle-message
	log.Info("HandlePeer: to for-loop", "peer", peer)

looping:
	for {
		err = r.HandleMessageWrapper(peer)
		if err != nil {
			log.Error("HandlePeer: message handling failed", "e", err)
			break looping
		}
	}
	log.Info("HandlePeer: after for-loop", "peer", peer, "e", err)

	return err
}

/*
AddPeer adds a new peer. expected no user-id.
	1. validate peer as random.
	2. set peer type as random.
	3. check dial-entity
	4. if there is a corresponding entity for dial: identify peer.
*/
func (r *BaseRouter) AddNewPeer(peer *PttPeer) error {
	r.peerLock.Lock()
	defer r.peerLock.Unlock()

	// 1. validate peer as random.
	err := r.ValidatePeer(peer.GetID(), peer.UserID, PeerTypeRandom, true)
	if err != nil {
		return err
	}

	// 2. set peer type as random.
	err = r.SetPeerType(peer, PeerTypeRandom, false, true)
	if err != nil {
		return err
	}

	err = r.CheckDialEntityAndIdentifyPeer(peer)
	if err != nil {
		return err
	}

	return nil
}

func (r *BaseRouter) FinishIdentifyPeer(peer *PttPeer, isLocked bool, isResetPeerType bool) error {
	/*
		if !isLocked {
			p.peerLock.Lock()
			defer p.peerLock.Unlock()
		}
	*/

	log.Debug("FinishIdentifyPeer", "peer", peer, "userID", peer.UserID)

	if peer.UserID == nil {
		return ErrPeerUserID
	}

	if isResetPeerType {
		peer.IsRegistered = false
		r.SetPeerType(peer, PeerTypeRandom, true, isLocked)
	}

	if peer.IsRegistered {
		return nil
	}

	peerType, err := r.determinePeerTypeFromAllEntities(peer)
	if err != nil {
		return err
	}

	log.Debug("FinishIdentifyPeer: to SetupPeer", "peer", peer, "peerType", peerType)

	return r.SetupPeer(peer, peerType, isLocked)
}

func (r *BaseRouter) ResetPeerType(peer *PttPeer, isLocked bool, isForceReset bool) error {

	if peer.IsToClose {
		return ErrToClose
	}

	log.Debug("ResetPeerType", "peer", peer, "userID", peer.UserID)

	if peer.UserID == nil {
		return ErrPeerUserID
	}

	if isForceReset {
		r.SetPeerType(peer, PeerTypeRandom, true, isLocked)
	}

	peerType, err := r.determinePeerTypeFromAllEntities(peer)
	if err != nil {
		return err
	}

	err = r.addPeerKnownUserID(peer, peerType, isLocked)
	if err != nil {
		return err
	}

	return nil
}

func (r *BaseRouter) determinePeerTypeFromAllEntities(peer *PttPeer) (PeerType, error) {

	/*
		if !isLocked {
			p.peerLock.Lock()
			defer p.peerLock.Unlock()
		}
	*/

	r.entityLock.RLock()
	defer r.entityLock.RUnlock()

	// me
	if r.myEntity != nil && r.myEntity.MyPM().IsMyDevice(peer) {
		return PeerTypeMe, nil
	}

	// hub
	if r.IsHubPeer(peer) {
		return PeerTypeHub, nil
	}

	// important
	var pm ProtocolManager
	for _, entity := range r.entities {
		pm = entity.PM()
		if pm.IsImportantPeer(peer) {
			return PeerTypeImportant, nil
		}
	}

	// member
	for _, entity := range r.entities {
		pm = entity.PM()
		if pm.IsMemberPeer(peer) {
			return PeerTypeMember, nil
		}
	}

	// pending
	for _, entity := range r.entities {
		pm = entity.PM()
		if pm.IsPendingPeer(peer) {
			return PeerTypePending, nil
		}
	}

	// random
	return PeerTypeRandom, nil
}

func (r *BaseRouter) IsHubPeer(peer *PttPeer) bool {
	return false
}

/*
SetupPeer setup peer with known user-id and register to entities.
*/
func (r *BaseRouter) SetupPeer(peer *PttPeer, peerType PeerType, isLocked bool) error {
	/*
		if !isLocked {
			p.peerLock.Lock()
			defer p.peerLock.Unlock()
		}
	*/

	if peer.UserID == nil {
		return ErrPeerUserID
	}

	err := r.addPeerKnownUserID(peer, peerType, isLocked)
	if err != nil {
		return err
	}

	err = r.RegisterPeerToEntities(peer)
	if err != nil {
		return err
	}

	return nil
}

/*
AddPeerKnownUserID deals with peer already with user-id and the corresponding peer-type.
	1. validate-peer.
	2. setup peer.
*/
func (r *BaseRouter) addPeerKnownUserID(peer *PttPeer, peerType PeerType, isLocked bool) error {
	if !isLocked {
		r.peerLock.Lock()
		defer r.peerLock.Unlock()
	}

	err := r.ValidatePeer(peer.GetID(), peer.UserID, peerType, true)
	if err != nil {
		return err
	}

	return r.SetPeerType(peer, peerType, false, true)
}

/*
RemovePeer removes peer
	1. get reigsteredPeer
	2. unregister peer from entities
	3. unset peer type
	4. disconnect
*/
func (r *BaseRouter) RemovePeer(peer *PttPeer, isLocked bool) error {
	log.Info("RemovePeer: start", "peer", peer)

	peer.IsToClose = true

	/*
		if !isLocked {
			p.peerLock.Lock()
			defer p.peerLock.Unlock()
		}
	*/

	log.Info("RemovePeer: after lock", "peer", peer)

	// peerID := peer.GetID()

	/*
		registeredPeer := p.GetPeer(peerID, isLocked)
		if registeredPeer == nil {
			return nil
		}
	*/

	err := r.UnregisterPeerFromEntities(peer)
	if err != nil {
		log.Error("unable to unregister peer from entities", "peer", peer, "e", err)
	}

	err = r.UnsetPeerType(peer, isLocked)
	if err != nil {
		log.Error("unable to remove peer", "peer", peer, "e", err)
	}

	node := &discover.Node{ID: peer.ID()}
	r.server.RemovePeer(node)

	log.Info("RemovePeer: done", "peer", peer)

	return nil
}

/*
ValidatePeer validates peer
	1. no need to do anything with my device
	2. check repeated user-id
	3. check
*/
func (r *BaseRouter) ValidatePeer(nodeID *discover.NodeID, userID *types.PttID, peerType PeerType, isLocked bool) error {
	if !isLocked {
		r.peerLock.Lock()
		defer r.peerLock.Unlock()
	}

	// no need to do anything with peer-type-me
	if peerType == PeerTypeMe {
		return nil
	}

	// check repeated user-id
	if userID != nil {
		origNodeID, ok := r.userPeerMap[*userID]
		if ok && !reflect.DeepEqual(origNodeID, nodeID) {
			return ErrAlreadyRegistered
		}
	}

	// check max-peers
	lenMyPeers := len(r.myPeers)
	lenHubPeers := len(r.hubPeers)
	lenImportantPeers := len(r.importantPeers)
	lenMemberPeers := len(r.memberPeers)
	lenPendingPeers := len(r.pendingPeers)
	lenRandomPeers := len(r.randomPeers)

	if peerType == PeerTypeHub && lenHubPeers >= r.config.MaxHubPeers {
		return p2p.DiscTooManyPeers
	}

	if peerType == PeerTypeImportant && lenImportantPeers >= r.config.MaxImportantPeers {
		return p2p.DiscTooManyPeers
	}

	if peerType == PeerTypeMember && lenMemberPeers >= r.config.MaxMemberPeers {
		return p2p.DiscTooManyPeers
	}

	if peerType == PeerTypePending && lenPendingPeers >= r.config.MaxPendingPeers {
		return p2p.DiscTooManyPeers
	}

	if peerType == PeerTypeRandom && lenRandomPeers >= r.config.MaxRandomPeers {
		return p2p.DiscTooManyPeers
	}

	lenPeers := lenMyPeers + lenHubPeers + lenImportantPeers + lenMemberPeers + lenPendingPeers + lenRandomPeers
	if lenPeers >= r.config.MaxPeers {
		err := r.dropAnyPeer(peerType, true)
		if err != nil {
			return err
		}
	}

	return nil
}

/*
SetPeerType sets the peer to the new peer-type and set in router peer-map.
*/
func (r *BaseRouter) SetPeerType(peer *PttPeer, peerType PeerType, isForce bool, isLocked bool) error {
	if peer.IsToClose {
		return ErrToClose
	}

	if !isLocked {
		peer.LockPeerType.Lock()
		defer peer.LockPeerType.Unlock()

		r.peerLock.Lock()
		defer r.peerLock.Unlock()

	}

	origPeerType := peer.PeerType

	if !isForce && origPeerType >= peerType {
		return nil
	}

	peer.PeerType = peerType

	log.Debug("SetPeerType", "peer", peer, "origPeerType", origPeerType, "peerType", peerType)

	switch origPeerType {
	case PeerTypeMe:
		delete(r.myPeers, peer.ID())
	case PeerTypeHub:
		delete(r.hubPeers, peer.ID())
	case PeerTypeImportant:
		delete(r.importantPeers, peer.ID())
	case PeerTypeMember:
		delete(r.memberPeers, peer.ID())
	case PeerTypePending:
		delete(r.pendingPeers, peer.ID())
	case PeerTypeRandom:
		delete(r.randomPeers, peer.ID())
	}

	switch peerType {
	case PeerTypeMe:
		log.Debug("SetPeerType: set as myPeer", "peer", peer)
		r.myPeers[peer.ID()] = peer
	case PeerTypeHub:
		r.hubPeers[peer.ID()] = peer
	case PeerTypeImportant:
		r.importantPeers[peer.ID()] = peer
	case PeerTypeMember:
		r.memberPeers[peer.ID()] = peer
	case PeerTypePending:
		r.pendingPeers[peer.ID()] = peer
	case PeerTypeRandom:
		log.Debug("SetPeerType: set as randomPeers", "peer", peer)
		r.randomPeers[peer.ID()] = peer
	}

	if peer.UserID != nil {
		r.userPeerMap[*peer.UserID] = peer.GetID()
	}

	return nil
}

/*
UnsetPeerType unsets the peer from the router peer-map.
*/
func (r *BaseRouter) UnsetPeerType(peer *PttPeer, isLocked bool) error {
	if !isLocked {
		r.peerLock.Lock()
		defer r.peerLock.Unlock()
	}

	peerID := peer.ID()
	peerType := peer.PeerType

	var thePeer *PttPeer
	ok := false
	switch peerType {
	case PeerTypeMe:
		thePeer, ok = r.myPeers[peerID]
		if !ok || peer != thePeer {
			return ErrNotRegistered
		}
		delete(r.myPeers, peerID)
	case PeerTypeHub:
		thePeer, ok = r.hubPeers[peerID]
		if !ok || peer != thePeer {
			return ErrNotRegistered
		}
		delete(r.hubPeers, peerID)
	case PeerTypeImportant:
		thePeer, ok = r.importantPeers[peerID]
		if !ok || peer != thePeer {
			return ErrNotRegistered
		}
		delete(r.importantPeers, peerID)
	case PeerTypeMember:
		thePeer, ok = r.memberPeers[peerID]
		if !ok || peer != thePeer {
			return ErrNotRegistered
		}
		delete(r.memberPeers, peerID)
	case PeerTypePending:
		thePeer, ok = r.pendingPeers[peerID]
		if !ok || peer != thePeer {
			return ErrNotRegistered
		}
		delete(r.pendingPeers, peerID)
	case PeerTypeRandom:
		thePeer, ok = r.randomPeers[peerID]
		if !ok || peer != thePeer {
			return ErrNotRegistered
		}
		delete(r.randomPeers, peerID)
	}

	return nil
}

/*
RegisterPeerToEntities registers peer to all the existing entities (register-peer-to-router is already done in CheckPeerType / SetPeerType)
	register to all the existing entities.
*/
func (r *BaseRouter) RegisterPeerToEntities(peer *PttPeer) error {
	/*
		if !isLocked {
			p.peerLock.Lock()
			defer p.peerLock.Unlock()
		}
	*/

	log.Info("RegisterPeerToEntities: start", "peer", peer)

	// register to all the existing entities.
	r.entityLock.RLock()
	defer r.entityLock.RUnlock()

	if peer.IsRegistered {
		return nil
	}

	var pm ProtocolManager
	var err error
	var fitPeerType PeerType
	for _, entity := range r.entities {
		pm = entity.PM()
		fitPeerType = pm.GetPeerType(peer)

		if fitPeerType < PeerTypePending {
			continue
		}

		log.Info("RegisterPeerToEntities (in-for-loop): to RegisterPeer", "entity", pm.Entity().IDString(), "peer", peer, "fitPeerType", fitPeerType)
		err = pm.RegisterPeer(peer, fitPeerType, false)
		log.Info("RegisterPeerToEntities (in-for-loop): after RegisterPeer", "entity", pm.Entity().IDString(), "peer", peer, "e", err)
		if err != nil {
			log.Warn("RegisterPeerToEntities: unable to register peer to entity", "peer", peer, "entity", entity.Name(), "e", err)
		}
	}

	peer.IsRegistered = true

	log.Info("RegisterPeerToEntities: done", "peer", peer)

	return nil
}

func (r *BaseRouter) GetPeerByUserID(id *types.PttID, isLocked bool) (*PttPeer, error) {
	if !isLocked {
		r.peerLock.RLock()
		defer r.peerLock.RUnlock()
	}

	// hub-peers
	for _, peer := range r.hubPeers {
		if reflect.DeepEqual(peer.UserID, id) {
			return peer, nil
		}
	}

	// important-peers
	for _, peer := range r.importantPeers {
		if reflect.DeepEqual(peer.UserID, id) {
			return peer, nil
		}
	}

	// member-peers
	for _, peer := range r.memberPeers {
		if reflect.DeepEqual(peer.UserID, id) {
			return peer, nil
		}
	}

	// pending-peers
	for _, peer := range r.pendingPeers {
		if reflect.DeepEqual(peer.UserID, id) {
			return peer, nil
		}
	}

	// random-peers
	for _, peer := range r.randomPeers {
		if reflect.DeepEqual(peer.UserID, id) {
			return peer, nil
		}
	}

	return nil, types.ErrInvalidID
}

/*
UnregisterPeerFromEntities unregisters the peer from all the existing entities.
*/
func (r *BaseRouter) UnregisterPeerFromEntities(peer *PttPeer) error {
	/*
		if !isLocked {
			p.peerLock.Lock()
			defer p.peerLock.Unlock()
		}
	*/

	log.Info("UnregisterPeerFromEntities: start", "peer", peer)

	r.entityLock.RLock()
	defer r.entityLock.RUnlock()

	var pm ProtocolManager
	var err error
	for _, entity := range r.entities {
		pm = entity.PM()

		log.Debug("UnregisterPeerFromEntities (in-for-loop): to pm.UnregisterPeer", "entity", entity.IDString(), "peer", peer)
		err = pm.UnregisterPeer(peer, false, true, true)
		log.Debug("UnregisterPeerFromEntities (in-for-loop): after pm.UnregisterPeer", "e", err, "entity", entity.IDString(), "peer", peer)
		if err != nil && err != ErrNotRegistered {
			log.Warn("UnregisterPeerFromEntities: unable to unregister peer from entity", "peer", peer, "entity", entity.IDString(), "e", err)
		}
	}

	log.Info("UnregisterPeerFromEntities: done", "peer", peer)

	return nil
}

/*
GetPeer gets specific peer
*/
func (r *BaseRouter) GetPeer(id *discover.NodeID, isLocked bool) *PttPeer {
	if !isLocked {
		r.peerLock.RLock()
		defer r.peerLock.RUnlock()
	}

	peer := r.myPeers[*id]
	if peer != nil {
		log.Debug("GetPeer: got peer", "routerType", "myPeer", "peerType", peer.PeerType)
		return peer
	}

	peer = r.hubPeers[*id]
	if peer != nil {
		log.Debug("GetPeer: got peer", "routerType", "hubPeer", "peerType", peer.PeerType)
		return peer
	}

	peer = r.importantPeers[*id]
	if peer != nil {
		log.Debug("GetPeer: got peer", "routerType", "importantPeer", "peerType", peer.PeerType)
		return peer
	}

	peer = r.memberPeers[*id]
	if peer != nil {
		log.Debug("GetPeer: got peer", "routerType", "memberPeer", "peerType", peer.PeerType)
		return peer
	}

	peer = r.pendingPeers[*id]
	if peer != nil {
		log.Debug("GetPeer: got peer", "routerType", "pendingPeer", "peerType", peer.PeerType)
		return peer
	}

	peer = r.randomPeers[*id]
	if peer != nil {
		log.Debug("GetPeer: got peer", "routerType", "randomPeer", "peerType", peer.PeerType)
		return peer
	}

	return nil
}

/*
DropAnyPeer drops any peers at most with the peerType.
*/
func (r *BaseRouter) dropAnyPeer(peerType PeerType, isLocked bool) error {
	if !isLocked {
		r.peerLock.Lock()
		defer r.peerLock.Unlock()
	}

	log.Debug("dropAnyPeer: start", "peerType", peerType)
	if len(r.randomPeers) != 0 {
		return r.dropAnyPeerCore(r.randomPeers, true)
	}
	if peerType == PeerTypeRandom {
		return p2p.DiscTooManyPeers
	}

	if len(r.pendingPeers) != 0 {
		return r.dropAnyPeerCore(r.pendingPeers, true)
	}
	if peerType == PeerTypePending {
		return p2p.DiscTooManyPeers
	}

	if len(r.memberPeers) != 0 {
		return r.dropAnyPeerCore(r.memberPeers, true)
	}
	if peerType == PeerTypeMember {
		return p2p.DiscTooManyPeers
	}

	if len(r.importantPeers) != 0 {
		return r.dropAnyPeerCore(r.importantPeers, true)
	}

	if peerType == PeerTypeImportant {
		return p2p.DiscTooManyPeers
	}

	if len(r.hubPeers) != 0 {
		return r.dropAnyPeerCore(r.hubPeers, true)
	}
	if peerType == PeerTypeHub {
		return p2p.DiscTooManyPeers
	}

	return nil
}

func (r *BaseRouter) dropAnyPeerCore(peers map[discover.NodeID]*PttPeer, isLocked bool) error {

	if !isLocked {
		r.peerLock.Lock()
		defer r.peerLock.Unlock()
	}

	randIdx := mrand.Intn(len(peers))

	i := 0

looping:
	for _, peer := range peers {
		if i == randIdx {
			log.Info("dropAnyPeerCore: to disconnect", "peer", peer, "i", i)

			node := &discover.Node{ID: peer.ID()}
			r.server.RemovePeer(node)
			break looping
		}

		i++
	}

	return nil
}

/**********
 * Dail
 **********/

func (r *BaseRouter) AddDial(nodeID *discover.NodeID, opKey *common.Address, peerType PeerType, isAddPeer bool) error {
	peer := r.GetPeer(nodeID, false)

	if peer != nil && peer.UserID != nil {
		log.Debug("router.AddDial: already got peer userID", "userID", peer.UserID, "peerType", peer.PeerType, "peerType", peerType)

		// setup peer with high peer type and check all the entities.
		if peer.PeerType < peerType {
			r.ResetPeerType(peer, false, false)
		}

		// just do the specific entity
		entity, err := r.getEntityFromHash(opKey, &r.lockOps, r.ops)
		if err != nil {
			return err
		}

		entity.PM().RegisterPeer(peer, peerType, false)
		return nil
	}

	err := r.dialHist.Add(nodeID, opKey)
	if err != nil {
		return err
	}

	log.Debug("router.AddDial: to CheckDialEntityAndIdentifyPeer", "nodeID", nodeID, "peer", peer)
	if peer != nil {
		return r.CheckDialEntityAndIdentifyPeer(peer)
	}

	if !isAddPeer {
		return nil
	}

	node := discover.NewWebrtcNode(*nodeID)
	r.Server().AddPeer(node)

	return nil
}

func (r *BaseRouter) CheckDialEntityAndIdentifyPeer(peer *PttPeer) error {
	// 1. check dial-entity
	entity, err := r.checkDialEntity(peer)
	log.Debug("CheckDialEntityAndIdentifyPeer: after checkDialEntity", "entity", entity, "e", err)
	if err != nil {
		return err
	}

	// 2. identify peer
	if entity != nil {
		entity.PM().IdentifyPeer(peer)
		return nil
	}

	return nil
}

func (r *BaseRouter) checkDialEntity(peer *PttPeer) (Entity, error) {
	dialInfo := r.dialHist.Get(peer.GetID())
	if dialInfo == nil {
		return nil, nil
	}

	r.lockOps.RLock()
	defer r.lockOps.RUnlock()

	entityID := r.ops[*dialInfo.OpKey]
	if entityID == nil {
		return nil, nil
	}

	r.entityLock.RLock()
	defer r.entityLock.RUnlock()

	entity := r.entities[*entityID]

	return entity, nil
}

/**********
 * Misc
 **********/

func randomPttPeers(peers []*PttPeer) []*PttPeer {
	newPeers := make([]*PttPeer, len(peers))
	perm := mrand.Perm(len(peers))
	for i, v := range perm {
		newPeers[v] = peers[i]
	}
	return newPeers
}

func (r *BaseRouter) ClosePeers() {
	r.peerLock.RLock()
	defer r.peerLock.RUnlock()

	for _, peer := range r.myPeers {
		node := &discover.Node{ID: peer.ID()}
		r.server.RemovePeer(node)
		log.Debug("ClosePeers: disconnect", "peer", peer)
	}

	for _, peer := range r.hubPeers {
		node := &discover.Node{ID: peer.ID()}
		r.server.RemovePeer(node)
		log.Debug("ClosePeers: disconnect", "peer", peer)
	}

	for _, peer := range r.importantPeers {
		node := &discover.Node{ID: peer.ID()}
		r.server.RemovePeer(node)
		log.Debug("ClosePeers: disconnect", "peer", peer)
	}

	for _, peer := range r.memberPeers {
		node := &discover.Node{ID: peer.ID()}
		r.server.RemovePeer(node)
		log.Debug("ClosePeers: disconnect", "peer", peer)
	}

	for _, peer := range r.pendingPeers {
		node := &discover.Node{ID: peer.ID()}
		r.server.RemovePeer(node)
		log.Debug("ClosePeers: disconnect", "peer", peer)
	}

	for _, peer := range r.randomPeers {
		node := &discover.Node{ID: peer.ID()}
		r.server.RemovePeer(node)
		log.Debug("ClosePeers: disconnect", "peer", peer)
	}
}
