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

package account

import (
	"github.com/ailabstw/go-pttai-core/common/types"
	pkgservice "github.com/ailabstw/go-pttai-core/service"
)

func (pm *ProtocolManager) CreateNameCard(nameCard []byte) error {

	myID := pm.Ptt().GetMyEntity().GetID()

	if !pm.IsMaster(myID, false) {
		return types.ErrInvalidID
	}

	_, err := pm.CreateObject(
		nil,
		UserOpTypeCreateNameCard,

		pm.userOplogMerkle,

		pm.NewNameCard,
		pm.NewUserOplogWithTS,
		nil,

		pm.SetUserDB,
		pm.broadcastUserOplogsCore,
		pm.broadcastUserOplogCore,

		nil,
	)
	if err != nil {
		return err
	}
	return nil
}

func (pm *ProtocolManager) NewNameCard(theData pkgservice.CreateData) (pkgservice.Object, pkgservice.OpData, error) {

	myID := pm.Ptt().GetMyEntity().GetID()
	entityID := pm.Entity().GetID()

	ts, err := types.GetTimestamp()
	if err != nil {
		return nil, nil, err
	}

	opData := &UserOpCreateNameCard{}

	nameCard, err := NewNameCard(ts, myID, entityID, nil, types.StatusInit, nil)
	if err != nil {
		return nil, nil, err
	}
	pm.SetNameCardDB(nameCard)

	return nameCard, opData, nil
}
