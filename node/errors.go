// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package node

import (
	"errors"
	"fmt"
	"reflect"
	"syscall"
)

var (
	ErrDataDirUsed    = errors.New("datadir already used by another process")
	ErrNodeStopped    = errors.New("node not started")
	ErrNodeRunning    = errors.New("node already running")
	ErrServiceUnknown = errors.New("unknown service")

	ErrNodeRestart = errors.New("node restart")

	dataDirInUseErrnos = map[uint]bool{11: true, 32: true, 35: true}
)

func ConvertFileLockError(err error) error {
	if errno, ok := err.(syscall.Errno); ok && dataDirInUseErrnos[uint(errno)] {
		return ErrDataDirUsed
	}
	return err
}

// DuplicateRouterError is returned during Node startup if a registered service
// constructor returns a service of the same type that was already started.
type DuplicateRouterError struct {
	Kind reflect.Type
}

// Error generates a textual representation of the duplicate service error.
func (e *DuplicateRouterError) Error() string {
	return fmt.Sprintf("duplicate service: %v", e.Kind)
}

// StopError is returned if a Node fails to stop either any of its registered
// services or itself.
type StopError struct {
	Server  error
	Routers map[reflect.Type]error
}

// Error generates a textual representation of the stop error.
func (e *StopError) Error() string {
	return fmt.Sprintf("server: %v, services: %v", e.Server, e.Routers)
}
