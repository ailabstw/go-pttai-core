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
	"reflect"

	"github.com/ailabstw/go-pttai-core/log"
	"github.com/ailabstw/go-pttai-core/p2p/discover"
	"github.com/ethereum/go-ethereum/common"
)

/*
HandleMessageWrapper
*/
func (r *BaseRouter) HandleMessageWrapper(peer *PttPeer) error {
	log.Debug("HandleMessageWrapper: to ReadMsg", "peer", peer)
	msg, err := peer.RW().ReadMsg()
	log.Debug("HandleMessageWrapper: after ReadMsg", "e", err, "code", msg.Code, "size", msg.Size)
	if err != nil {
		log.Error("HandleMessageWrapper: unable ReadMsg", "peer", peer, "e", err)
		return err
	}
	defer msg.Discard()

	if msg.Size > ProtocolMaxMsgSize {
		log.Error("HandleMessageWrapper: exceed size", "peer", peer, "msg.Size", msg.Size)
		return ErrMsgTooLarge
	}

	data := &RouterData{}
	err = msg.Decode(data)
	if err != nil {
		log.Error("HandleMessageWrapper: unable to decode data", "peer", peer, "e", err)
		return err
	}

	err = r.HandleMessage(CodeType(msg.Code), data, peer)
	if err != nil {
		log.Error("HandleMessageWrapper: unable to handle-msg", "code", msg.Code, "e", err, "peer", peer)
		return err
	}

	return nil
}

/*
HandleMessage handles message
*/
func (r *BaseRouter) HandleMessage(code CodeType, data *RouterData, peer *PttPeer) error {
	var err error

	log.Debug("HandleMessage: start", "code", code, "peer", peer, "peerType", peer.PeerType)

	if !reflect.DeepEqual(data.Node, discover.EmptyNodeID) && !reflect.DeepEqual(data.Node, r.myNodeID[:]) {
		log.Error("HandleMessage: the msg is not for me or not for broadcast", "code", code, "data.Node", data.Node, "peer", peer)
		return ErrInvalidData
	}

	evCode, evHash, encData, err := r.UnmarshalData(data)
	if err != nil {
		log.Error("HandleMessage: unable to unmarshal", "data", data, "e", err)
		return err
	}

	if evCode != code || (code < CodeTypeRequireHash && !reflect.DeepEqual(evHash[:], data.Hash[:])) {
		log.Error("HandleMessage: hash not match", "evHash", evHash, "dataHash", data.Hash)
		return ErrInvalidData
	}

	switch code {
	case CodeTypeJoin:
		err = r.HandleCodeJoin(evHash, encData, peer)
	case CodeTypeJoinAck:
		err = r.HandleCodeJoinAck(evHash, encData, peer)

	case CodeTypeOp:
		err = r.HandleCodeOp(evHash, encData, peer)
	case CodeTypeOpFail:
		err = r.HandleCodeOpFail(evHash, encData, peer)

	case CodeTypeRequestOpKey:
		err = r.HandleCodeRequestOpKey(evHash, encData, peer)
	case CodeTypeRequestOpKeyFail:
		err = r.HandleCodeRequestOpKeyFail(evHash, encData, peer)
	case CodeTypeRequestOpKeyAck:
		err = r.HandleCodeRequestOpKeyAck(evHash, encData, peer)

	case CodeTypeEntityDeleted:
		err = r.HandleCodeEntityDeleted(evHash, encData, peer)

	case CodeTypeOpCheckMember:
		err = r.HandleCodeOpCheckMember(evHash, encData, peer)
	case CodeTypeOpCheckMemberAck:
		err = r.HandleCodeOpCheckMemberAck(evHash, encData, peer)

	case CodeTypeIdentifyPeer:
		err = r.HandleCodeIdentifyPeer(evHash, encData, peer)
	case CodeTypeIdentifyPeerFail:
		err = r.HandleCodeIdentifyPeerFail(evHash, encData, peer)
	case CodeTypeIdentifyPeerWithMyID:
		err = r.HandleCodeIdentifyPeerWithMyID(evHash, encData, peer)
	case CodeTypeIdentifyPeerWithMyIDChallenge:
		err = r.HandleCodeIdentifyPeerWithMyIDChallenge(evHash, encData, peer)
	case CodeTypeIdentifyPeerWithMyIDChallengeAck:
		err = r.HandleCodeIdentifyPeerWithMyIDChallengeAck(evHash, encData, peer)
	case CodeTypeIdentifyPeerWithMyIDAck:
		err = r.HandleCodeIdentifyPeerWithMyIDAck(evHash, encData, peer)
	default:
		err = ErrInvalidMsgCode
	}

	if err != nil {
		log.Error("Ptt.HandleMessage", "code", code, "e", err)

	}

	return nil
}

func (r *BaseRouter) HandleCodeJoin(hash *common.Address, encData []byte, peer *PttPeer) error {
	entity, err := r.getEntityFromHash(hash, &r.lockJoins, r.joins)
	if err != nil {
		log.Error("HandleCodeJoin: getEntityFromHash", "e", err)
		return err
	}

	pm := entity.PM()
	keyInfo, err := pm.GetJoinKeyFromHash(hash)
	if err != nil {
		log.Error("HandleCodeJoin: unable to get JoinKeyInfo", "hash", hash, "e", err)
		return err
	}

	op, dataBytes, err := r.DecryptData(encData, keyInfo)
	if err != nil {
		log.Error("HandleCodeJoin: unable to DecryptData", "e", err)
		return err
	}

	log.Debug("HandleCodeJoin: start", "op", op, "joinMsg", JoinMsg, "joinEntityMsg", JoinEntityMsg)

	switch op {
	case JoinMsg:
		err = r.HandleJoin(dataBytes, hash, entity, pm, keyInfo, peer)
	case JoinEntityMsg:
		err = r.HandleJoinEntity(dataBytes, hash, entity, pm, keyInfo, peer)
	default:
		err = ErrInvalidMsgCode
	}

	return err
}

func (r *BaseRouter) HandleCodeJoinAck(hash *common.Address, encData []byte, peer *PttPeer) error {

	joinRequest, err := r.myEntity.GetJoinRequest(hash)
	log.Debug("HandleCodeJoinAck: after GetJoinRequest", "e", err)
	if err != nil {
		return err
	}

	keyInfo := joinKeyToKeyInfo(joinRequest.Key)

	op, dataBytes, err := r.DecryptData(encData, keyInfo)
	if err != nil {
		return err
	}

	log.Debug("HandleCodeJoinAck: start", "op", op, "ApproveJoinMsg", ApproveJoinMsg)

	switch op {
	case JoinAckChallengeMsg:
		err = r.HandleJoinAckChallenge(dataBytes, hash, joinRequest, peer)
	case ApproveJoinMsg:
		err = r.HandleApproveJoin(dataBytes, hash, joinRequest, peer)
	default:
		err = ErrInvalidMsgCode
	}

	return err
}

func (r *BaseRouter) HandleCodeOp(hash *common.Address, encData []byte, peer *PttPeer) error {

	entity, err := r.getEntityFromHash(hash, &r.lockOps, r.ops)
	log.Debug("HandleCodeOp: after getEntityFromHash", "e", err, "hash", hash)
	if err != nil {
		log.Error("HandleCodeOp: invalid entity", "hash", hash, "e", err)
		return r.OpFail(hash, peer)
	}

	pm := entity.PM()

	err = PMHandleMessageWrapper(pm, hash, encData, peer)

	return err
}

func (r *BaseRouter) HandleCodeOpFail(hash *common.Address, encData []byte, peer *PttPeer) error {

	return r.HandleOpFail(encData, peer)
}

func (r *BaseRouter) HandleCodeRequestOpKey(hash *common.Address, encData []byte, peer *PttPeer) error {

	return r.HandleRequestOpKey(encData, peer)
}

func (r *BaseRouter) HandleCodeRequestOpKeyFail(hash *common.Address, encData []byte, peer *PttPeer) error {

	return r.HandleRequestOpKeyFail(encData, peer)
}

func (r *BaseRouter) HandleCodeRequestOpKeyAck(hash *common.Address, encData []byte, peer *PttPeer) error {
	return r.HandleRequestOpKeyAck(encData, peer)
}

func (r *BaseRouter) HandleCodeEntityDeleted(hash *common.Address, encData []byte, peer *PttPeer) error {
	return r.HandleEntityTerminal(encData, peer)
}

func (r *BaseRouter) HandleCodeOpCheckMember(hash *common.Address, encData []byte, peer *PttPeer) error {
	return r.HandleOpCheckMember(encData, peer)
}

func (r *BaseRouter) HandleCodeOpCheckMemberAck(hash *common.Address, encData []byte, peer *PttPeer) error {
	return r.HandleOpCheckMemberAck(encData, peer)
}

func (r *BaseRouter) HandleCodeIdentifyPeer(hash *common.Address, encData []byte, peer *PttPeer) error {

	entity, err := r.getEntityFromHash(hash, &r.lockOps, r.ops)
	if err != nil {
		log.Error("HandleCodeIdentifyPeer: invalid entity", "hash", hash, "e", err)
		return r.IdentifyPeerFail(hash, peer)
	}

	pm := entity.PM()

	err = PMHandleMessageWrapper(pm, hash, encData, peer)
	if err != nil {
		r.IdentifyPeerFail(hash, peer)
	}

	return err
}

func (r *BaseRouter) HandleCodeIdentifyPeerFail(hash *common.Address, encData []byte, peer *PttPeer) error {

	return r.HandleIdentifyPeerFail(encData, peer)
}

func (r *BaseRouter) HandleCodeIdentifyPeerWithMyID(hash *common.Address, encData []byte, peer *PttPeer) error {

	return r.HandleIdentifyPeerWithMyID(encData, peer)
}

func (r *BaseRouter) HandleCodeIdentifyPeerWithMyIDChallenge(hash *common.Address, encData []byte, peer *PttPeer) error {

	return r.HandleIdentifyPeerWithMyIDChallenge(encData, peer)
}

func (r *BaseRouter) HandleCodeIdentifyPeerWithMyIDChallengeAck(hash *common.Address, encData []byte, peer *PttPeer) error {

	return r.HandleIdentifyPeerWithMyIDChallengeAck(encData, peer)
}

func (r *BaseRouter) HandleCodeIdentifyPeerWithMyIDAck(hash *common.Address, encData []byte, peer *PttPeer) error {

	return r.HandleIdentifyPeerWithMyIDAck(encData, peer)
}
