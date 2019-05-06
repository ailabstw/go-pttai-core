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
	"github.com/ailabstw/go-pttai-core/account"
	"github.com/ailabstw/go-pttai-core/common/types"
	pkgservice "github.com/ailabstw/go-pttai-core/service"
)

/*
BackendGetFriend provides just UserName and nothing else for saving the bandwidth
*/
type BackendGetFriend struct {
	ID       *types.PttID
	FriendID *types.PttID
	Name     []byte

	BoardID         *types.PttID
	Status          types.Status
	ArticleCreateTS types.Timestamp //
	LastSeen        types.Timestamp
}

func friendToBackendGetFriend(f *Friend, userName *account.UserName) *BackendGetFriend {
	messageCreateTS := f.MessageCreateTS
	if messageCreateTS.IsLess(f.CreateTS) {
		messageCreateTS = f.CreateTS
	}

	lastSeen := f.LastSeen
	if lastSeen.IsLess(f.CreateTS) {
		lastSeen = f.CreateTS
	}

	return &BackendGetFriend{
		ID:              f.ID,
		FriendID:        f.FriendID,
		Name:            userName.Name,
		Status:          f.Status,
		BoardID:         f.BoardID,
		ArticleCreateTS: messageCreateTS,
		LastSeen:        f.LastSeen,
	}
}

type BackendCreateMessage struct {
	FriendID  *types.PttID
	MessageID *types.PttID
	BlockID   *types.PttID
	NBlock    int
}

func messageToBackendCreateMessage(m *Message) *BackendCreateMessage {

	return &BackendCreateMessage{
		FriendID:  m.EntityID,
		MessageID: m.ID,
		BlockID:   m.BlockInfo.ID,
		NBlock:    m.BlockInfo.NBlock,
	}
}

type BackendGetMessage struct {
	ID        *types.PttID
	CreateTS  types.Timestamp //
	UpdateTS  types.Timestamp //
	CreatorID *types.PttID    //
	FriendID  *types.PttID    //
	BlockID   *types.PttID    //
	NBlock    int             //
	Status    types.Status
}

func messageToBackendGetMessage(m *Message) *BackendGetMessage {

	return &BackendGetMessage{
		ID:        m.ID,
		CreateTS:  m.CreateTS,
		UpdateTS:  m.UpdateTS,
		CreatorID: m.CreatorID,
		FriendID:  m.EntityID,
		BlockID:   m.BlockInfo.ID,
		NBlock:    m.BlockInfo.NBlock,
		Status:    m.Status,
	}
}

type BackendMessageBlock struct {
	V         types.Version
	ID        *types.PttID
	MessageID *types.PttID
	ObjID     *types.PttID
	BlockID   uint32

	Status types.Status

	CreateTS types.Timestamp
	UpdateTS types.Timestamp

	CreatorID *types.PttID
	UpdaterID *types.PttID

	Buf [][]byte
}

func contentBlockToBackendMessageBlock(msg *Message, blockInfoID *types.PttID, contentBlock *pkgservice.ContentBlock) *BackendMessageBlock {

	objID := msg.ID
	return &BackendMessageBlock{
		V:         types.CurrentVersion,
		ID:        blockInfoID,
		MessageID: objID,
		ObjID:     objID,
		BlockID:   contentBlock.BlockID,
		Status:    msg.Status,

		CreateTS: msg.CreateTS,
		UpdateTS: msg.UpdateTS,

		CreatorID: msg.CreatorID,
		UpdaterID: msg.UpdaterID,

		Buf: contentBlock.Buf,
	}
}
