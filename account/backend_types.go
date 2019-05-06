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

import "github.com/ailabstw/go-pttai-core/common/types"

type BackendUserName struct {
	ID   *types.PttID
	Name []byte
}

func userNameToBackendUserName(u *UserName) *BackendUserName {
	return &BackendUserName{
		ID:   u.ID,
		Name: u.Name,
	}
}

type BackendUserImg struct {
	ID     *types.PttID
	Type   ImgType
	Img    string
	Width  uint16
	Height uint16
}

func userImgToBackendUserImg(u *UserImg) *BackendUserImg {
	return &BackendUserImg{
		ID:     u.ID,
		Type:   u.ImgType,
		Img:    u.Str, //XXX TODO: ensure the img
		Width:  u.Width,
		Height: u.Height,
	}
}

type BackendNameCard struct {
	ID   *types.PttID
	Card []byte
}

func userNameToBackendNameCard(u *NameCard) *BackendNameCard {
	return &BackendNameCard{
		ID:   u.ID,
		Card: u.Card,
	}
}
