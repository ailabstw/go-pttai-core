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
	"github.com/ailabstw/go-pttai-core/p2p/discover"
	"github.com/ethereum/go-ethereum/common"
)

/*
me (my_info) for all the entities.
*/
type MyEntity interface {
	GetID() *types.PttID
	GetStatus() types.Status

	PM() ProtocolManager

	Name() string

	NewOpKeyInfo(entityID *types.PttID, setOpKeyObjDB func(k *KeyInfo)) (*KeyInfo, error)

	GetProfile() Entity

	GetNodeSignID() *types.PttID

	GetUserNodeID(id *types.PttID) (*discover.NodeID, error)

	Sign(oplog *BaseOplog) error
	InternalSign(oplog *BaseOplog) error
	MasterSign(oplog *BaseOplog) error
	SignBlock(block *Block) error

	IsValidInternalOplog(signInfos []*SignInfo) (*types.PttID, uint32, bool)

	CreateEntityOplog(entity Entity) error
	CreateJoinEntityOplog(entity Entity) error

	GetValidateKey() *types.PttID
}

/*
MyEntity for ptt.
*/
type RouterMyEntity interface {
	MyEntity

	MyPM() MyProtocolManager

	SignKey() *KeyInfo

	// join
	GetJoinRequest(hash *common.Address) (*JoinRequest, error)
	HandleApproveJoin(dataBytes []byte, hash *common.Address, joinRequest *JoinRequest, peer *PttPeer) error

	// node
	GetLenNodes() int

	Service() Service
}
