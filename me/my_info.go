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

package me

import (
	"crypto/ecdsa"
	"encoding/json"
	"reflect"

	"github.com/ailabstw/go-pttai-core/account"
	"github.com/ailabstw/go-pttai-core/common/types"
	"github.com/ailabstw/go-pttai-core/friend"
	"github.com/ailabstw/go-pttai-core/log"
	"github.com/ailabstw/go-pttai-core/p2p/discover"
	pkgservice "github.com/ailabstw/go-pttai-core/service"
	"github.com/ethereum/go-ethereum/common"
)

type MyInfo struct {
	*pkgservice.BaseEntity `json:"e"`

	UpdateTS types.Timestamp `json:"UT"`

	ProfileID *types.PttID     `json:"PID"`
	Profile   *account.Profile `json:"-"`

	signKeyInfo     *pkgservice.KeyInfo
	nodeSignKeyInfo *pkgservice.KeyInfo

	NodeSignID *types.PttID `json:"-"`

	myKey   *ecdsa.PrivateKey
	nodeKey *ecdsa.PrivateKey

	validateKey *types.PttID
}

func NewEmptyMyInfo() *MyInfo {
	return &MyInfo{BaseEntity: &pkgservice.BaseEntity{SyncInfo: &pkgservice.BaseSyncInfo{}}}
}

func NewMyInfo(id *types.PttID, myKey *ecdsa.PrivateKey, router pkgservice.MyRouter, service pkgservice.Service, spm pkgservice.ServiceProtocolManager, dbLock *types.LockMap) (*MyInfo, error) {
	ts, err := types.GetTimestamp()
	if err != nil {
		return nil, err
	}

	e := pkgservice.NewBaseEntity(id, ts, id, types.StatusPending, dbMe, dbLock)

	m := &MyInfo{
		BaseEntity: e,
		UpdateTS:   ts,
		myKey:      myKey,
	}

	// new my node
	myNodeID := router.MyNodeID()
	myNode, err := NewMyNode(ts, id, myNodeID, 1)
	if err != nil {
		return nil, err
	}

	myNode.Status = types.StatusAlive
	myNode.NodeType = router.MyNodeType()

	_, err = myNode.Save()
	if err != nil {
		return nil, err
	}

	err = m.Init(router, service, spm)
	if err != nil {
		return nil, err
	}

	return m, nil
}

func (m *MyInfo) GetUpdateTS() types.Timestamp {
	return m.UpdateTS
}

func (m *MyInfo) SetUpdateTS(ts types.Timestamp) {
	m.UpdateTS = ts
}

func (m *MyInfo) Init(router pkgservice.Router, service pkgservice.Service, spm pkgservice.ServiceProtocolManager) error {

	log.Debug("me.Init: start")
	myRouter, ok := router.(pkgservice.MyRouter)
	if !ok {
		return pkgservice.ErrInvalidData
	}

	MyID := spm.(*ServiceProtocolManager).MyID
	m.SetDB(dbMe, spm.GetDBLock())

	err := m.InitPM(myRouter, service)
	if err != nil {
		return err
	}

	myID := m.ID
	nodeKey := myRouter.MyNodeKey()
	nodeID := myRouter.MyNodeID()
	nodeSignID, err := setNodeSignID(nodeID, myID)

	m.nodeKey = nodeKey
	m.NodeSignID = nodeSignID

	m.validateKey, err = types.NewPttID()
	if err != nil {
		return err
	}

	// my-key
	if m.myKey == nil {
		m.myKey, err = m.loadMyKey()
		if err != nil {
			if !reflect.DeepEqual(myID, MyID) {
				return nil
			}
			return err
		}
	}

	// sign-key
	err = m.CreateSignKeyInfo()
	if err != nil {
		return err
	}

	err = m.CreateNodeSignKeyInfo()
	if err != nil {
		return err
	}

	// set my entity
	if !reflect.DeepEqual(myID, MyID) {
		return nil
	}

	// profile
	accountSPM := service.(*Backend).accountBackend.SPM()
	if m.ProfileID != nil {
		profile := accountSPM.Entity(m.ProfileID)
		if profile == nil {
			return pkgservice.ErrInvalidEntity
		}
		m.Profile = profile.(*account.Profile)
	}

	myRouter.SetMyEntity(m)

	return nil
}

func (m *MyInfo) loadMyKey() (*ecdsa.PrivateKey, error) {
	cfg := m.Service().(*Backend).Config

	return cfg.GetDataPrivateKeyByID(m.ID)
}

func (m *MyInfo) InitPM(router pkgservice.MyRouter, service pkgservice.Service) error {
	pm, err := NewProtocolManager(m, router, service)
	if err != nil {
		log.Error("InitPM: unable to NewProtocolManager", "e", err)
		return err
	}

	m.BaseEntity.Init(pm, router, service)

	return nil
}

func (m *MyInfo) MarshalKey() ([]byte, error) {
	key := append(DBMePrefix, m.ID[:]...)

	return key, nil
}

func (m *MyInfo) Marshal() ([]byte, error) {
	return json.Marshal(m)
}

func (m *MyInfo) Unmarshal(theBytes []byte) error {
	err := json.Unmarshal(theBytes, m)
	if err != nil {
		return err
	}

	// postprocess

	return nil
}

func (m *MyInfo) Save(isLocked bool) error {
	if !isLocked {
		err := m.Lock()
		if err != nil {
			return err
		}
		defer m.Unlock()
	}

	key, err := m.MarshalKey()
	if err != nil {
		return err
	}

	marshaled, err := m.Marshal()
	if err != nil {
		return err
	}

	err = dbMeCore.Put(key, marshaled)
	if err != nil {
		return err
	}

	return nil
}

func (m *MyInfo) MustSave(isLocked bool) error {
	if !isLocked {
		m.MustLock()
		defer m.Unlock()
	}

	key, err := m.MarshalKey()
	if err != nil {
		return err
	}

	marshaled, err := m.Marshal()
	if err != nil {
		return err
	}

	err = dbMeCore.Put(key, marshaled)
	if err != nil {
		return err
	}

	return nil

}

func (m *MyInfo) GetJoinRequest(hash *common.Address) (*pkgservice.JoinRequest, error) {
	return m.PM().(*ProtocolManager).GetJoinRequest(hash)
}

func (m *MyInfo) GetLenNodes() int {
	return len(m.PM().(*ProtocolManager).MyNodes)
}

func (m *MyInfo) IsValidInternalOplog(signInfos []*pkgservice.SignInfo) (*types.PttID, uint32, bool) {
	return m.PM().(*ProtocolManager).IsValidInternalOplog(signInfos)
}

func (m *MyInfo) MyPM() pkgservice.MyProtocolManager {
	return m.PM().(*ProtocolManager)
}

func (m *MyInfo) GetMasterKey() *ecdsa.PrivateKey {
	return m.myKey
}

func (m *MyInfo) GetValidateKey() *types.PttID {
	return m.validateKey
}

func (m *MyInfo) GetProfile() pkgservice.Entity {
	return m.Profile
}

func (m *MyInfo) GetUserNodeID(id *types.PttID) (*discover.NodeID, error) {
	friendBackend := m.Service().(*Backend).friendBackend

	theFriend, err := friendBackend.SPM().(*friend.ServiceProtocolManager).GetFriendEntityByFriendID(id)
	if err != nil {
		return nil, err
	}
	if theFriend == nil {
		return nil, types.ErrInvalidID
	}

	friendPM := theFriend.PM().(*friend.ProtocolManager)

	return friendPM.GetUserNodeID()
}
