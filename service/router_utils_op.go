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
	"github.com/ailabstw/go-pttai-core/log"
	"github.com/ethereum/go-ethereum/common"
)

func (r *BaseRouter) AddOpKey(hash *common.Address, entityID *types.PttID, isLocked bool) error {
	if !isLocked {
		r.LockOps()
		defer r.UnlockOps()
	}

	log.Debug("AddOpkey: to add key", "hash", hash, "entityID", entityID)

	r.ops[*hash] = entityID

	return nil
}

func (r *BaseRouter) RemoveOpKey(hash *common.Address, entityID *types.PttID, isLocked bool) error {
	if !isLocked {
		r.LockOps()
		defer r.UnlockOps()
	}

	log.Debug("RemoveOpKey: to remove key", "hash", hash, "entityID", entityID)

	delete(r.ops, *hash)

	return nil
}

func (r *BaseRouter) LockOps() {
	r.lockOps.Lock()
}

func (r *BaseRouter) UnlockOps() {
	r.lockOps.Unlock()
}

func (r *BaseRouter) RemoveOpHash(hash *common.Address) error {
	entityID, ok := r.ops[*hash]
	if !ok {
		return nil
	}

	entity, ok := r.entities[*entityID]
	if !ok {
		return r.RemoveOpKey(hash, entityID, false)
	}

	return entity.PM().RemoveOpKeyFromHash(hash, false, true, true)
}
