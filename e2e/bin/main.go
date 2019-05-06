// usage: ./bin <ID> <PORT>
package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ailabstw/go-pttai-core/account"
	"github.com/ailabstw/go-pttai-core/friend"
	"github.com/ailabstw/go-pttai-core/log"
	"github.com/ailabstw/go-pttai-core/me"
	"github.com/ailabstw/go-pttai-core/node"
	"github.com/ailabstw/go-pttai-core/p2p/discover"
	"github.com/ailabstw/go-pttai-core/service"
)

func main() {
	idArg := os.Args[1]
	id, err := strconv.Atoi(idArg)
	if err != nil {
		panic(err)
	}

	portArg := os.Args[2]
	port, err := strconv.Atoi(portArg)
	if err != nil {
		panic(err)
	}
	fmt.Printf("test node %v, %v\n", id, port)

	os.MkdirAll(fmt.Sprintf("./tmp/test/%d", id), 0755)
	log.Root().SetHandler(log.Must.FileHandler(fmt.Sprintf("./tmp/test/%d/log.tmp.txt", id), log.TerminalFormat(false)))

	n, err := prepareNode(id, port)
	if err != nil {
		panic(err)
	}

	err = n.Start()
	if err != nil {
		panic(err)
	}

	for {
	}
}

func prepareNode(id int, port int) (*node.Node, error) {
	nodeCfg := &node.DefaultConfig
	nodeCfg.DataDir = fmt.Sprintf("./tmp/test/%d/", id)
	nodeCfg.HTTPHost = "127.0.0.1"
	nodeCfg.HTTPPort = port
	nodeCfg.IPCPath = ""
	nodeCfg.P2P.MaxPeers = 100
	signalServerURL, err := url.Parse("ws://127.0.0.1:9489/signal")
	if err != nil {
		return nil, err
	}
	nodeCfg.P2P.SignalServerURL = *signalServerURL

	meConfig := &me.DefaultConfig
	meConfig.DataDir = filepath.Join(nodeCfg.DataDir, "me")
	err = meConfig.SetMyKey("", "", "", false)
	if err != nil {
		return nil, err
	}

	routerConfig := &service.DefaultConfig
	routerConfig.DataDir = filepath.Join(fmt.Sprintf("./tmp/test/%d/", id), "service")

	accountConfig := &account.DefaultConfig
	accountConfig.DataDir = filepath.Join(fmt.Sprintf("./tmp/test/%d/", id), "account")

	friendConfig := &friend.DefaultConfig
	friendConfig.DataDir = filepath.Join(fmt.Sprintf("./tmp/test/%d/", id), "friend")
	friendConfig.MinSyncRandomSeconds = 5
	friendConfig.MaxSyncRandomSeconds = 7

	n, err := node.New(nodeCfg)
	if err != nil {
		return nil, err
	}

	n.Register(func(ctx *service.RouterContext) (service.NodeRouter, error) {
		nodeKey := nodeCfg.NodeKey()
		nodeID := discover.PubkeyID(&nodeKey.PublicKey)

		ptt, err := service.NewRouter(ctx, routerConfig, &nodeID, nodeKey)
		if err != nil {
			return nil, err
		}
		accountBackend, err := account.NewBackend(ctx, accountConfig, ptt)
		if err != nil {
			return nil, err
		}
		err = ptt.RegisterService(accountBackend)
		if err != nil {
			return nil, err
		}

		// friend
		friendBackend, err := friend.NewBackend(ctx, friendConfig, meConfig.ID, ptt, accountBackend)
		if err != nil {
			return nil, err
		}
		err = ptt.RegisterService(friendBackend)
		if err != nil {
			return nil, err
		}

		// me
		meBackend, err := me.NewBackend(ctx, meConfig, ptt, accountBackend, friendBackend)
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

	return n, nil
}
