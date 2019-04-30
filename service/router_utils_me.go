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

	"github.com/ailabstw/go-pttai-core/common/types"
	"github.com/ailabstw/go-pttai-core/p2p/discover"
)

/**********
 * Me
 **********/

func (r *BaseRouter) MyNodeID() *discover.NodeID {
	return r.myNodeID
}

func (r *BaseRouter) MyRaftID() uint64 {
	return r.myRaftID
}

func (r *BaseRouter) MyNodeType() NodeType {
	return r.myNodeType
}

func (r *BaseRouter) MyNodeKey() *ecdsa.PrivateKey {
	return r.myNodeKey
}

func (r *BaseRouter) SetMyEntity(myEntity RouterMyEntity) error {
	r.myEntity = myEntity
	r.myService = myEntity.Service()

	return nil
}

func (r *BaseRouter) GetMyEntity() MyEntity {
	return r.myEntity
}

func (r *BaseRouter) GetMyEntityFromMe(myID *types.PttID) Entity {
	return nil
}

func (r *BaseRouter) GetMyService() Service {
	return r.myService
}
