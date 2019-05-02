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
	"crypto/ecdsa"
	"sync"

	"github.com/ailabstw/go-pttai-core/common/types"
	"github.com/ailabstw/go-pttai-core/log"
	"github.com/ailabstw/go-pttai-core/p2p"
	"github.com/ailabstw/go-pttai-core/p2p/discover"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
)

/*
Router is the public-access version of Router.
*/
type Router interface {
	// event-mux

	ErrChan() *types.Chan

	// peers
	IdentifyPeer(entityID *types.PttID, quitSync chan struct{}, peer *PttPeer, isForce bool) (*IdentifyPeer, error)
	IdentifyPeerAck(challenge *types.Salt, peer *PttPeer) (*IdentifyPeerAck, error)
	HandleIdentifyPeerAck(entityID *types.PttID, data *IdentifyPeerAck, peer *PttPeer) error

	FinishIdentifyPeer(peer *PttPeer, isLocked bool, isResetPeerType bool) error

	ResetPeerType(peer *PttPeer, isLocked bool, isResetPeerType bool) error

	NoMorePeers() chan struct{}

	AddDial(nodeID *discover.NodeID, opKey *common.Address, peerType PeerType, isAddPeer bool) error

	// entities

	RegisterEntity(e Entity, isLocked bool, isPeerLock bool) error
	UnregisterEntity(e Entity, isLocked bool) error

	RegisterEntityPeerWithOtherUserID(e Entity, id *types.PttID, peerType PeerType, isLocked bool) error

	// join

	AddJoinKey(hash *common.Address, entityID *types.PttID, isLocked bool) error
	RemoveJoinKey(hash *common.Address, entityID *types.PttID, isLocked bool) error

	TryJoin(challenge []byte, hash *common.Address, key *ecdsa.PrivateKey, request *JoinRequest) error

	// op

	AddOpKey(hash *common.Address, entityID *types.PttID, isLocked bool) error
	RemoveOpKey(hash *common.Address, entityID *types.PttID, isLocked bool) error
	RequestOpKeyByEntity(entity Entity, peer *PttPeer) error

	// sync

	SyncWG() *sync.WaitGroup

	// me

	MyNodeID() *discover.NodeID

	GetMyEntity() MyEntity
	GetMyService() Service

	// data

	EncryptData(op OpType, data []byte, keyInfo *KeyInfo) ([]byte, error)
	DecryptData(ciphertext []byte, keyInfo *KeyInfo) (OpType, []byte, error)

	MarshalData(code CodeType, hash *common.Address, encData []byte) (*RouterData, error)
	UnmarshalData(pttData *RouterData) (CodeType, *common.Address, []byte, error)
}

/*
NodeRouter is the interface for ptt as the service in the node-level.
*/
type NodeRouter interface {
	// Protocols retrieves the P2P protocols the service wishes to start.
	Protocols() []p2p.Protocol

	// APIs retrieves the list of RPC descriptors the service provides
	APIs() []rpc.API

	// Start is called after all services have been constructed and the networking
	// layer was also initialized to spawn any goroutines required by the service.
	Start(server *p2p.Server) error

	// Stop terminates all goroutines belonging to the service, blocking until they
	// are all terminated.
	Stop() error
}

/*
MyRouter is Router interface for me (my_info).
*/
type MyRouter interface {
	Router

	// event-mux

	NotifyNodeRestart() *types.Chan
	NotifyNodeStop() *types.Chan

	// MyEntity

	SetMyEntity(m RouterMyEntity) error
	MyRaftID() uint64
	MyNodeType() NodeType
	MyNodeKey() *ecdsa.PrivateKey

	// SetPeerType

	SetPeerType(peer *PttPeer, peerType PeerType, isForce bool, isLocked bool) error
	SetupPeer(peer *PttPeer, peerType PeerType, isLocked bool) error

	GetEntities() map[types.PttID]Entity
}

type BaseRouter struct {
	config *Config

	// event-mux
	eventMux *event.TypeMux

	notifyNodeRestart *types.Chan
	notifyNodeStop    *types.Chan
	errChan           *types.Chan

	// peers
	peerLock sync.RWMutex

	myPeers        map[discover.NodeID]*PttPeer
	hubPeers       map[discover.NodeID]*PttPeer
	importantPeers map[discover.NodeID]*PttPeer
	memberPeers    map[discover.NodeID]*PttPeer
	pendingPeers   map[discover.NodeID]*PttPeer
	randomPeers    map[discover.NodeID]*PttPeer

	userPeerMap map[types.PttID]*discover.NodeID

	noMorePeers chan struct{}

	peerWG sync.WaitGroup

	dialHist *DialHistory

	// entities
	entityLock sync.RWMutex

	entities map[types.PttID]Entity

	// joins
	lockJoins sync.RWMutex
	joins     map[common.Address]*types.PttID

	lockConfirmJoin sync.RWMutex
	confirmJoins    map[string]*ConfirmJoin

	// ops
	lockOps sync.RWMutex
	ops     map[common.Address]*types.PttID

	// sync
	quitSync chan struct{}
	syncWG   sync.WaitGroup

	// services
	services map[string]Service

	// p2p server
	server *p2p.Server

	// protocols
	protocols []p2p.Protocol

	// apis
	apis []rpc.API

	// network-id
	networkID uint32

	// me
	myEntity   RouterMyEntity
	myNodeID   *discover.NodeID // ptt knows only my-node-id
	myRaftID   uint64
	myNodeType NodeType
	myNodeKey  *ecdsa.PrivateKey
	myService  Service
}

func NewRouter(ctx *RouterContext, cfg *Config, myNodeID *discover.NodeID, myNodeKey *ecdsa.PrivateKey) (*BaseRouter, error) {
	// init-service
	InitService(cfg.DataDir)

	myRaftID, err := myNodeID.ToRaftID()
	if err != nil {
		return nil, err
	}

	r := &BaseRouter{
		config: cfg,

		myNodeID:   myNodeID,
		myRaftID:   myRaftID,
		myNodeType: cfg.NodeType,
		myNodeKey:  myNodeKey,

		// event-mux
		eventMux: new(event.TypeMux),

		notifyNodeRestart: types.NewChan(1),
		notifyNodeStop:    types.NewChan(1),

		// peer
		noMorePeers: make(chan struct{}),

		myPeers:        make(map[discover.NodeID]*PttPeer),
		hubPeers:       make(map[discover.NodeID]*PttPeer),
		importantPeers: make(map[discover.NodeID]*PttPeer),
		memberPeers:    make(map[discover.NodeID]*PttPeer),
		pendingPeers:   make(map[discover.NodeID]*PttPeer),
		randomPeers:    make(map[discover.NodeID]*PttPeer),

		userPeerMap: make(map[types.PttID]*discover.NodeID),

		dialHist: NewDialHistory(),

		// entities
		entities: make(map[types.PttID]Entity),

		// joins
		joins:        make(map[common.Address]*types.PttID),
		confirmJoins: make(map[string]*ConfirmJoin),

		// ops
		ops: make(map[common.Address]*types.PttID),

		// sync
		quitSync: make(chan struct{}),

		// services
		services: make(map[string]Service),

		errChan: types.NewChan(1),
	}

	r.apis = r.routerAPIs()

	r.protocols = r.generateProtocols()

	return r, nil
}

/**********
 * NodeRouter
 **********/

func (r *BaseRouter) Protocols() []p2p.Protocol {
	return r.protocols
}

func (r *BaseRouter) APIs() []rpc.API {
	return r.apis
}

func (r *BaseRouter) Prestart() error {
	var err error
	errMap := make(map[string]error)
	for name, service := range r.services {
		err = service.Prestart()
		if err != nil {
			errMap[name] = err
			break
		}
	}

	if len(errMap) != 0 {
		return errMapToErr(errMap)
	}

	return nil
}

func (r *BaseRouter) Start(server *p2p.Server) error {
	r.server = server

	// Start services
	var err error
	successMap := make(map[string]Service)
	errMap := make(map[string]error)

	myService := r.myService
	if myService != nil {
		err = myService.Start()
		if err != nil {
			errMap["me"] = err
		} else {
			successMap["me"] = myService
		}
	}

	if err == nil {
		for name, service := range r.services {
			if service == myService {
				continue
			}
			log.Info("Start: to start service", "name", name)
			err = service.Start()
			if err != nil {
				errMap[name] = err
				break
			}
			successMap[name] = service
		}
	}

	if err != nil {
		for name, successService := range successMap {
			err = successService.Stop()
			if err != nil {
				errMap[name] = err
			}
		}
	}
	if len(errMap) != 0 {
		return errMapToErr(errMap)
	}

	return nil
}

func (r *BaseRouter) Stop() error {
	close(r.quitSync)
	close(r.noMorePeers)

	// close all service-loop
	errMap := make(map[string]error)
	for name, service := range r.services {
		err := service.Stop()
		if err != nil {
			errMap[name] = err
		}
	}

	log.Debug("Stop: to wait syncWG")

	r.syncWG.Wait()

	// close peers
	r.ClosePeers()

	log.Debug("Stop: to wait peerWG")

	r.peerWG.Wait()

	// remove ptt-level chan

	r.eventMux.Stop()

	log.Debug("Stop: done")

	if len(errMap) != 0 {
		return errMapToErr(errMap)
	}

	return nil
}

/**********
 * RW
 **********/

func (r *BaseRouter) RWInit(peer *PttPeer, version uint) {
	if rw, ok := peer.RW().(MeteredMsgReadWriter); ok {
		rw.Init(version)
	}
}

/**********
 * Service
 **********/

/*
RegisterService registers service into ptt.
*/
func (r *BaseRouter) RegisterService(service Service) error {
	log.Info("RegisterService", "name", service.Name())
	r.apis = append(r.apis, service.APIs()...)

	name := service.Name()

	r.services[name] = service

	log.Info("RegisterService: done", "name", service.Name())

	return nil
}

/**********
 * Chan
 **********/

func (r *BaseRouter) NotifyNodeRestart() *types.Chan {
	return r.notifyNodeRestart
}

func (r *BaseRouter) NotifyNodeStop() *types.Chan {
	return r.notifyNodeStop
}

func (r *BaseRouter) ErrChan() *types.Chan {
	return r.errChan
}

/**********
 * Server
 **********/

func (r *BaseRouter) Server() *p2p.Server {
	return r.server
}

/**********
 * Peer
 **********/

func (r *BaseRouter) NoMorePeers() chan struct{} {
	return r.noMorePeers
}
