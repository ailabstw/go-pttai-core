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
	"github.com/ailabstw/go-pttai-core/pttdb"
)

func (pm *BaseProtocolManager) GetMemberList(startID *types.PttID, limit int, listOrder pttdb.ListOrder, isLocked bool) ([]*Member, error) {
	obj := NewEmptyMember()
	pm.SetMemberObjDB(obj)

	objs, err := GetObjList(obj, startID, limit, listOrder, isLocked)
	if err != nil {
		return nil, err
	}
	typedObjs := ObjsToMembers(objs)

	return typedObjs, nil
}
