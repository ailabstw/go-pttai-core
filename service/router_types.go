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
	"github.com/ailabstw/go-pttai-core/p2p/discover"
)

// RouterEventData
type RouterEventData struct {
	Code    CodeType `json:"C"`
	Hash    []byte   `json:"H,omitempty"`
	EncData []byte   `json:"D,omitempty"`
}

// RouterData
type RouterData struct {
	Node       []byte   `json:"N,omitempty"`
	Code       CodeType `json:"C"`
	Hash       []byte   `json:"H,omitempty"`
	EvWithSalt []byte   `json:"E,omitempty"`
	Checksum   []byte   `json:"c,omitempty"`

	Relay uint8 `json:"R"`
}

func (r *RouterData) Clone() *RouterData {
	return &RouterData{
		Node:       r.Node,
		Code:       r.Code,
		Hash:       r.Hash,
		EvWithSalt: r.EvWithSalt,
		Checksum:   r.Checksum,
		Relay:      r.Relay,
	}
}

// RouterStatus
type RouterStatus struct {
	Version   uint32
	NetworkID uint32
}

// PttPeerInfo
type PttPeerInfo struct {
	NodeID   *discover.NodeID `json:"N"`
	UserID   *types.PttID     `json:"U"`
	PeerType PeerType         `json:"T"`
}

type NodeRouterInfo struct {
	NodeID *discover.NodeID `json:"N"`
	UserID *types.PttID     `json:"U"`

	Peers    int `json:"NP"`
	Entities int `json:"NE"`
	Services int `json:"NS"`
}
