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
	"github.com/ailabstw/go-pttai-core/log"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func joinKeyToKeyInfo(key *ecdsa.PrivateKey) *KeyInfo {
	return &KeyInfo{
		Key:         key,
		KeyBytes:    crypto.FromECDSA(key),
		PubKeyBytes: crypto.FromECDSAPub(&key.PublicKey),
	}
}

func (r *BaseRouter) AddJoinKey(hash *common.Address, entityID *types.PttID, isLocked bool) error {
	if !isLocked {
		r.LockJoins()
		defer r.UnlockJoins()
	}

	log.Debug("AddJoinKey: start", "hash", hash, "entityID", entityID)

	r.joins[*hash] = entityID

	return nil
}

func (r *BaseRouter) RemoveJoinKey(hash *common.Address, entityID *types.PttID, isLocked bool) error {
	if !isLocked {
		r.LockJoins()
		defer r.UnlockJoins()
	}

	log.Debug("RemoveJoinKey: start", "hash", hash, "entityID", entityID)

	delete(r.joins, *hash)

	return nil
}

func (r *BaseRouter) LockJoins() {
	r.lockJoins.Lock()
}

func (r *BaseRouter) UnlockJoins() {
	r.lockJoins.Unlock()
}
