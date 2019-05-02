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
	"github.com/ailabstw/go-pttai-core/account"
	"github.com/ailabstw/go-pttai-core/friend"
	"github.com/ailabstw/go-pttai-core/log"
	pkgservice "github.com/ailabstw/go-pttai-core/service"
	"github.com/ethereum/go-ethereum/rpc"
)

type Backend struct {
	*pkgservice.BaseService

	Config *Config

	accountBackend *account.Backend
	friendBackend  *friend.Backend

	myRouter pkgservice.MyRouter
}

func NewBackend(ctx *pkgservice.RouterContext, cfg *Config, router pkgservice.MyRouter, accountBackend *account.Backend, friendBacked *friend.Backend) (*Backend, error) {
	err := InitMe(cfg.DataDir)
	if err != nil {
		return nil, err
	}

	backend := &Backend{
		Config:   cfg,
		myRouter: router,

		accountBackend: accountBackend,
		friendBackend:  friendBacked,
	}

	spm, err := NewServiceProtocolManager(cfg.ID, router, backend)
	if err != nil {
		return nil, err
	}

	svc, err := pkgservice.NewBaseService(router, spm)
	if err != nil {
		return nil, err
	}
	backend.BaseService = svc

	if spm.MyInfo != nil {
		return backend, nil
	}

	err = spm.CreateMe(cfg.ID, cfg.PrivateKey)
	if err != nil {
		log.Debug("me.NewBackend: unable to CreateMe", "e", err)
		return nil, err
	}

	return backend, nil
}

func (b *Backend) Start() error {
	b.SPM().(*ServiceProtocolManager).Start()
	return nil
}

func (b *Backend) Stop() error {
	b.SPM().(*ServiceProtocolManager).Stop()

	log.Debug("Stop: to TeardownMe")

	TeardownMe()

	log.Debug("Stop: after TeardownMe")

	return nil
}

func (b *Backend) APIs() []rpc.API {
	return []rpc.API{
		{
			Namespace: "me",
			Version:   "1.0",
			Service:   NewPrivateAPI(b),
			Public:    pkgservice.IsPrivateAsPublic,
		},
		{
			Namespace: "me",
			Version:   "1.0",
			Service:   NewPublicAPI(b),
			Public:    true,
		},
	}
}

func (b *Backend) Name() string {
	return "me"
}
