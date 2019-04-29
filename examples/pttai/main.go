// Copyright 2019 The go-pttai Authors
// This file is part of go-pttai.
//
// go-pttai is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-pttai is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-pttai. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"os"
	"os/signal"
	"os/user"
	"path"
	"reflect"
	"strconv"
	"strings"
	"syscall"

	"github.com/ailabstw/go-pttai-core/account"
	"github.com/ailabstw/go-pttai-core/friend"
	"github.com/ailabstw/go-pttai-core/log"
	"github.com/ailabstw/go-pttai-core/me"
	"github.com/ailabstw/go-pttai-core/node"
	"github.com/ailabstw/go-pttai-core/p2p"
	"github.com/ailabstw/go-pttai-core/p2p/discover"
	"github.com/ailabstw/go-pttai-core/service"
	cli "gopkg.in/urfave/cli.v1"
)

type Config struct {
	Node    *node.Config
	Me      *me.Config
	Account *account.Config
	Friend  *friend.Config
	Ptt     *service.Config
	Utils   *UtilsConfig
}

type UtilsConfig struct {
	HTTPDir        string
	HTTPAddr       string
	ExternHTTPAddr string
}

func main() {
	gptt(&cli.Context{})
}

func gptt(ctx *cli.Context) error {
	log.Info("PTT.ai: Hello world!")

	cfg := Config{
		Node:    &node.Config{P2P: p2p.Config{}},
		Me:      &me.Config{},
		Account: &account.Config{},
		Friend:  &friend.Config{},
		Ptt:     &service.Config{},
		Utils:   &UtilsConfig{},
	}
	cfg.Utils.ExternHTTPAddr = "http://localhost:9776"
	cfg.Node.HTTPHost = ""
	cfg.Node.HTTPPort = 9450

	// new node
	n, err := node.New(cfg.Node)
	if err != nil {
		return err
	}

	// register ptt
	if err := registerPtt(n, &cfg); err != nil {
		return err
	}

	// node start
	if err := n.Start(); err != nil {
		return err
	}

	// set-signal
	go setSignal(n)

	// wait-node
	if err := WaitNode(n); err != nil {
		return err
	}

	log.Info("PTT.ai: see u later～")

	return nil
}

func registerPtt(n *node.Node, cfg *Config) error {
	return n.Register(func(ctx *service.ServiceContext) (service.PttService, error) {
		myNodeKey := cfg.Node.NodeKey()
		myNodeID := discover.PubkeyID(&myNodeKey.PublicKey)

		ptt, err := service.NewPtt(ctx, cfg.Ptt, &myNodeID, myNodeKey)
		if err != nil {
			return nil, err
		}

		accountBackend, err := account.NewBackend(ctx, cfg.Account, ptt)
		if err != nil {
			return nil, err
		}
		err = ptt.RegisterService(accountBackend)
		if err != nil {
			return nil, err
		}

		// friend
		friendBackend, err := friend.NewBackend(ctx, cfg.Friend, cfg.Me.ID, ptt, accountBackend)
		if err != nil {
			return nil, err
		}
		err = ptt.RegisterService(friendBackend)
		if err != nil {
			return nil, err
		}

		// me
		meBackend, err := me.NewBackend(ctx, cfg.Me, ptt, accountBackend, friendBackend)
		if err != nil {
			return nil, err
		}

		err = ptt.RegisterService(meBackend)
		if err != nil {
			return nil, err
		}

		err = ptt.Prestart()
		if err != nil {
			log.Error("unable to do Prestart", "e", err)
			return nil, err
		}

		return ptt, nil
	})
}

func setSignal(n *node.Node) {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigc)

	<-sigc

	log.Debug("setSignal: received break-signal")
	for i := 10; i > 0; i-- {
		<-sigc
		if i > 1 {
			log.Warn("Already shutting down, interrupt more to panic.", "times", i-1)
		}
	}
	panic("boom")
}

func WaitNode(n *node.Node) error {
	log.Info("start Waiting...")

	ptt := n.Services()[reflect.TypeOf(&service.BasePtt{})].(*service.BasePtt)

loop:
	for {
		select {
		case _, ok := <-ptt.NotifyNodeRestart().GetChan():
			log.Debug("WaitNode: NotifyNodeRestart: start")
			if !ok {
				break loop
			}
			err := n.Restart(false, true)
			if err != nil {
				return err
			}
			ptt = n.Services()[reflect.TypeOf(&service.BasePtt{})].(*service.BasePtt)
			log.Debug("WaitNode: NotifyNodeRestart: done")
		case _, ok := <-ptt.NotifyNodeStop().GetChan():
			log.Debug("WaitNode: NotifyNodeStop: start")
			if !ok {
				break loop
			}
			n.Stop(false, false)
			log.Debug("WaitNode: NotifyNodeStop: done")
			break loop
		case err, ok := <-ptt.ErrChan().GetChan():
			if !ok {
				break loop
			}
			log.Error("Received err from ptt", "e", err)
			break loop
		case err, ok := <-n.StopChan:
			log.Debug("WaitNode: StopChan: start")
			if ok && err != nil {
				log.Error("Wait", "e", err)
				return err
			}
			log.Debug("WaitNode: StopChan: done")
			break loop
		}
	}

	return nil
}

// SetContentConfig applies node-related command line flags to the config.
func SetUtilsConfig(ctx *cli.Context, cfg *UtilsConfig) {
	switch {
	case ctx.GlobalIsSet(HTTPDirFlag.Name):
		cfg.HTTPDir = ctx.GlobalString(HTTPDirFlag.Name)
	}

	switch {
	case ctx.GlobalIsSet(HTTPAddrFlag.Name):
		cfg.HTTPAddr = ctx.GlobalString(HTTPAddrFlag.Name)
	}

	switch {
	case ctx.GlobalIsSet(ExternHTTPAddrFlag.Name):
		cfg.ExternHTTPAddr = ctx.GlobalString(ExternHTTPAddrFlag.Name)
	default:
		cfg.ExternHTTPAddr = "http://" + cfg.HTTPAddr
	}

}

var (
	// HTTP server
	HTTPAddrFlag = cli.StringFlag{
		Name:  "httpaddr",
		Usage: "HTTP server listening addr",
	}
	HTTPDirFlag = cli.StringFlag{
		Name:  "httpdir",
		Usage: "HTTP server serving file-dir",
	}
	ExternHTTPAddrFlag = cli.StringFlag{
		Name:  "exthttpaddr",
		Usage: "External HTTP server listening addr",
	}
	DataDirFlag = DirectoryFlag{
		Name:  "datadir",
		Usage: "Data directory for the databases and keystore",
		Value: DirectoryString{node.DefaultDataDir()},
	}
	KeyStoreDirFlag = DirectoryFlag{
		Name:  "keystore",
		Usage: "Directory for the keystore (default = inside the datadir)",
	}
	// RPC settings
	RPCEnabledFlag = cli.BoolTFlag{
		Name:  "rpc",
		Usage: "Enable the HTTP-RPC server",
	}
	RPCListenAddrFlag = cli.StringFlag{
		Name:  "rpcaddr",
		Usage: "HTTP-RPC server listening interface",
		Value: node.DefaultHTTPHost,
	}
	RPCPortFlag = cli.IntFlag{
		Name:  "rpcport",
		Usage: "HTTP-RPC server listening port",
		Value: node.DefaultHTTPPort,
	}
	RPCCORSDomainFlag = cli.StringFlag{
		Name:  "rpccorsdomain",
		Usage: "Comma separated list of domains from which to accept cross origin requests (browser enforced)",
		Value: "",
	}
	RPCVirtualHostsFlag = cli.StringFlag{
		Name:  "rpcvhosts",
		Usage: "Comma separated list of virtual hostnames from which to accept requests (server enforced). Accepts '*' wildcard.",
		Value: strings.Join(node.DefaultConfig.HTTPVirtualHosts, ","),
	}
	RPCApiFlag = cli.StringFlag{
		Name:  "rpcapi",
		Usage: "API's offered over the HTTP-RPC interface",
		Value: "",
	}
	ExternRPCAddrFlag = cli.StringFlag{
		Name:  "extrpcaddr",
		Usage: "External HTTP-RPC server listening addr",
	}
)

// SetNodeConfig applies node-related command line flags to the config.
func SetNodeConfig(ctx *cli.Context, cfg *node.Config) {
	log.Debug("SetNodeConfig: start")
	setHTTP(ctx, cfg)

	// data-dir
	switch {
	case ctx.GlobalIsSet(DataDirFlag.Name):
		cfg.DataDir = ctx.GlobalString(DataDirFlag.Name)
	}

	if ctx.GlobalIsSet(KeyStoreDirFlag.Name) {
		cfg.KeyStoreDir = ctx.GlobalString(KeyStoreDirFlag.Name)
	}
}

func setHTTP(ctx *cli.Context, cfg *node.Config) {
	if ctx.GlobalBool(RPCEnabledFlag.Name) && cfg.HTTPHost == "" {
		cfg.HTTPHost = "127.0.0.1"
		if ctx.GlobalIsSet(RPCListenAddrFlag.Name) {
			cfg.HTTPHost = ctx.GlobalString(RPCListenAddrFlag.Name)
		}
	}

	if ctx.GlobalIsSet(RPCPortFlag.Name) {
		cfg.HTTPPort = ctx.GlobalInt(RPCPortFlag.Name)
	}
	if ctx.GlobalIsSet(RPCCORSDomainFlag.Name) {
		cfg.HTTPCors = splitAndTrim(ctx.GlobalString(RPCCORSDomainFlag.Name))
	}
	if ctx.GlobalIsSet(RPCApiFlag.Name) {
		cfg.HTTPModules = splitAndTrim(ctx.GlobalString(RPCApiFlag.Name))
	}
	if ctx.GlobalIsSet(RPCVirtualHostsFlag.Name) {
		cfg.HTTPVirtualHosts = splitAndTrim(ctx.GlobalString(RPCVirtualHostsFlag.Name))
	}

	if ctx.GlobalIsSet(ExternRPCAddrFlag.Name) {
		cfg.ExternHTTPAddr = ctx.GlobalString(ExternRPCAddrFlag.Name)
	} else {
		cfg.ExternHTTPAddr = "http://" + cfg.HTTPHost + ":" + strconv.Itoa(cfg.HTTPPort)
	}
}

func splitAndTrim(input string) []string {
	result := strings.Split(input, ",")
	for i, r := range result {
		result[i] = strings.TrimSpace(r)
	}
	return result
}

// Custom cli.Flag type which expand the received string to an absolute path.
// e.g. ~/.ethereum -> /home/username/.ethereum
type DirectoryFlag struct {
	Name  string
	Value DirectoryString
	Usage string
}

// Custom type which is registered in the flags library which cli uses for
// argument parsing. This allows us to expand Value to an absolute path when
// the argument is parsed
type DirectoryString struct {
	Value string
}

func (self *DirectoryString) String() string {
	return self.Value
}

func (self *DirectoryString) Set(value string) error {
	self.Value = expandPath(value)
	return nil
}

// Expands a file path
// 1. replace tilde with users home dir
// 2. expands embedded environment variables
// 3. cleans the path, e.g. /a/b/../c -> /a/c
// Note, it has limitations, e.g. ~someuser/tmp will not be expanded
func expandPath(p string) string {
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, "~\\") {
		if home := homeDir(); home != "" {
			p = home + p[1:]
		}
	}
	return path.Clean(os.ExpandEnv(p))
}

func homeDir() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	if usr, err := user.Current(); err == nil {
		return usr.HomeDir
	}
	return ""
}
