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

package api

import (
    "net"
    "net/http"

    "github.com/ethereum/go-ethereum/rpc"
)

// StartHTTPEndpoint starts the HTTP RPC endpoint, configured with cors/vhosts/modules
func StartHTTPEndpoint(endpoint string, apis []rpc.API, modules []string, cors []string, vhosts []string, timeouts rpc.HTTPTimeouts) (net.Listener, *rpc.Server, *http.Server, error) {
    // Generate the whitelist based on the allowed modules
    whitelist := make(map[string]bool)
    for _, module := range modules {
        whitelist[module] = true
    }
    // Register all the APIs exposed by the services
    handler := rpc.NewServer()
    for _, api := range apis {
        if whitelist[api.Namespace] || (len(whitelist) == 0 && api.Public) {
            if err := handler.RegisterName(api.Namespace, api.Service); err != nil {
                return nil, nil, nil, err
            }
        }
    }
    // All APIs registered, start the HTTP listener
    var (
        listener net.Listener
        err      error
    )
    if listener, err = net.Listen("tcp", endpoint); err != nil {
        return nil, nil, nil, err
    }
    httpServer := rpc.NewHTTPServer(cors, vhosts, timeouts, handler)
    go httpServer.Serve(listener)
    return listener, handler, httpServer, err
}
