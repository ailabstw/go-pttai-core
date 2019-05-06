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

package friend

import (
	"github.com/ailabstw/go-pttai-core/common/types"
	"github.com/ailabstw/go-pttai-core/log"
	"github.com/ailabstw/go-pttai-core/pttdb"
	pkgservice "github.com/ailabstw/go-pttai-core/service"
)

type ApproveJoin struct {
	Friend *Friend

	Oplog0     *pkgservice.BaseOplog
	MasterLogs []*pkgservice.BaseOplog
	MemberLogs []*pkgservice.BaseOplog
	OpKey      *pkgservice.KeyInfo
	OpKeyLog   *pkgservice.BaseOplog
}

func (pm *ProtocolManager) ApproveJoinFriend(joinEntity *pkgservice.JoinEntity, keyInfo *pkgservice.KeyInfo, peer *pkgservice.PttPeer) (*pkgservice.KeyInfo, interface{}, error) {

	// friend
	f := pm.Entity().(*Friend)

	// master
	oplog := &pkgservice.BaseOplog{}
	pm.SetMasterDB(oplog)
	masterLogs, err := pkgservice.GetOplogList(oplog, nil, 0, pttdb.ListOrderNext, types.StatusAlive, false)
	if err != nil {
		log.Error("ApproveJoinFriend: unable to get master logs", "e", err, "entity", pm.Entity().GetID())
		return nil, nil, err
	}

	log.Debug("ApproveJoinFriend: after master GetOplogList", "masterLogs", masterLogs)

	// member
	_, _, err = pm.AddMember(joinEntity.ID, true)
	if err != nil {
		log.Error("ApproveJoinFriend: unable to add member", "e", err, "entity", pm.Entity().GetID())
		return nil, nil, err
	}

	pm.SetMemberDB(oplog)
	memberLogs, err := pkgservice.GetOplogList(oplog, nil, 0, pttdb.ListOrderNext, types.StatusAlive, false)
	if err != nil {
		log.Error("ApproveJoinFriend: unable to get member logs", "e", err, "entity", pm.Entity().GetID())
		return nil, nil, err
	}

	// op-key
	opKey, err := pm.GetNewestOpKey(false)
	if err != nil {
		return nil, nil, err
	}

	opKeyLog := &pkgservice.BaseOplog{}
	pm.SetOpKeyDB(opKeyLog)
	err = opKeyLog.Get(opKey.LogID, false)
	if err != nil {
		return nil, nil, err
	}

	approveJoin := &ApproveJoin{
		Friend: f,

		Oplog0:     pm.GetOplog0(),
		MasterLogs: masterLogs,
		MemberLogs: memberLogs,

		OpKey:    opKey,
		OpKeyLog: opKeyLog,
	}
	return opKey, approveJoin, nil
}
