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
	"github.com/ailabstw/go-pttai-core/pttdb"
	"github.com/ethereum/go-ethereum/common"
)

func (r *BaseRouter) GetVersion() (string, error) {
	return r.config.Version, nil
}

func (r *BaseRouter) GetGitCommit() (string, error) {
	return r.config.GitCommit, nil
}

func (r *BaseRouter) Shutdown() (bool, error) {
	log.Debug("Shutdown: start")
	r.notifyNodeStop.PassChan(struct{}{})
	log.Debug("Shutdown: done")
	return true, nil
}

func (r *BaseRouter) Restart() (bool, error) {
	r.notifyNodeRestart.PassChan(struct{}{})
	return true, nil
}

/**********
 * Peer
 **********/

func (r *BaseRouter) CountPeers() (*BackendCountPeers, error) {
	r.peerLock.RLock()
	defer r.peerLock.RUnlock()

	return &BackendCountPeers{
		MyPeers:        len(r.myPeers),
		ImportantPeers: len(r.importantPeers),
		MemberPeers:    len(r.memberPeers),
		RandomPeers:    len(r.randomPeers),
	}, nil
}

func (r *BaseRouter) BEGetPeers() ([]*BackendPeer, error) {
	r.peerLock.RLock()
	defer r.peerLock.RUnlock()

	peerList := make([]*BackendPeer, 0, len(r.myPeers)+len(r.importantPeers)+len(r.memberPeers)+len(r.randomPeers))

	var backendPeer *BackendPeer
	for _, peer := range r.myPeers {
		backendPeer = PeerToBackendPeer(peer)
		peerList = append(peerList, backendPeer)
	}

	for _, peer := range r.importantPeers {
		backendPeer = PeerToBackendPeer(peer)
		peerList = append(peerList, backendPeer)
	}

	for _, peer := range r.memberPeers {
		backendPeer = PeerToBackendPeer(peer)
		peerList = append(peerList, backendPeer)
	}

	for _, peer := range r.randomPeers {
		backendPeer = PeerToBackendPeer(peer)
		peerList = append(peerList, backendPeer)
	}

	return peerList, nil
}

/**********
 * Entities
 **********/

func (r *BaseRouter) CountEntities() (int, error) {
	return len(r.entities), nil
}

/**********
 * Join
 **********/

func (r *BaseRouter) GetJoins() map[common.Address]*types.PttID {
	return r.joins
}

func (r *BaseRouter) GetConfirmJoins() ([]*BackendConfirmJoin, error) {
	r.lockConfirmJoin.RLock()
	defer r.lockConfirmJoin.RUnlock()

	results := make([]*BackendConfirmJoin, len(r.confirmJoins))

	i := 0
	for _, confirmJoin := range r.confirmJoins {
		backendConfirmJoin := &BackendConfirmJoin{
			ID:         confirmJoin.JoinEntity.ID,
			Name:       confirmJoin.JoinEntity.Name,
			EntityID:   confirmJoin.Entity.GetID(),
			EntityName: []byte(confirmJoin.Entity.Name()),
			UpdateTS:   confirmJoin.UpdateTS,
			NodeID:     confirmJoin.Peer.GetID(),
			JoinType:   confirmJoin.JoinType,
		}
		results[i] = backendConfirmJoin

		i++
	}

	return results, nil
}

/**********
 * Op
 **********/

func (r *BaseRouter) GetOps() map[common.Address]*types.PttID {
	return r.ops
}

/**********
 * PttOplog
 **********/

func (r *BaseRouter) BEGetPttOplogList(logIDBytes []byte, limit int, listOrder pttdb.ListOrder) ([]*PttOplog, error) {

	logID, err := types.UnmarshalTextPttID(logIDBytes, true)
	if err != nil {
		return nil, err
	}

	return r.GetPttOplogList(logID, limit, listOrder, types.StatusAlive)
}

func (r *BaseRouter) MarkPttOplogSeen() (types.Timestamp, error) {
	ts, err := types.GetTimestamp()
	if err != nil {
		return types.ZeroTimestamp, err
	}

	tsBytes, err := ts.Marshal()
	if err != nil {
		return types.ZeroTimestamp, err
	}

	err = dbMeta.Put(DBPttLogSeenPrefix, tsBytes)
	if err != nil {
		return types.ZeroTimestamp, err
	}

	return ts, nil
}

func (r *BaseRouter) GetPttOplogSeen() (types.Timestamp, error) {
	tsBytes, err := dbMeta.Get(DBPttLogSeenPrefix)
	if err != nil {
		return types.ZeroTimestamp, nil
	}

	ts, err := types.UnmarshalTimestamp(tsBytes)
	if err != nil {
		return types.ZeroTimestamp, nil
	}

	return ts, nil
}

func (r *BaseRouter) GetLastAnnounceP2PTS() (types.Timestamp, error) {
	return types.TimeToTimestamp(r.server.LastAnnounceP2PTS), nil
}
