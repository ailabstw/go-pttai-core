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

type Config struct {
	Node    *node.Config
	Me      *me.Config
	Account *account.Config
	Friend  *friend.Config
	Router  *service.Config
	Utils   *UtilsConfig
}

type UtilsConfig struct {
	HTTPDir        string
	HTTPAddr       string
	ExternHTTPAddr string
}

func prepareNode(id int, port int) (*node.Node, error) {
	cfg := Config{
		Node:    &node.DefaultConfig,
		Me:      &me.DefaultConfig,
		Account: &account.DefaultConfig,
		Friend:  &friend.DefaultConfig,
		Router:  &service.DefaultConfig,
		Utils:   &UtilsConfig{},
	}
	cfg.Utils.ExternHTTPAddr = fmt.Sprintf("http://localhost:%d", 9776+id)
	cfg.Node.DataDir = fmt.Sprintf("./tmp/test/%d/", id)
	cfg.Node.HTTPHost = "127.0.0.1"
	cfg.Node.HTTPPort = port
	cfg.Node.IPCPath = ""
	cfg.Node.P2P.MaxPeers = 100
	signalServerURL, err := url.Parse("ws://127.0.0.1:9489/signal")
	if err != nil {
		return nil, err
	}
	cfg.Node.P2P.SignalServerURL = *signalServerURL
	cfg.Me.DataDir = filepath.Join(cfg.Node.DataDir, "me")
	cfg.Router.DataDir = filepath.Join(fmt.Sprintf("./tmp/test/%d/", id), "service")
	cfg.Account.DataDir = filepath.Join(fmt.Sprintf("./tmp/test/%d/", id), "account")
	cfg.Friend.DataDir = filepath.Join(fmt.Sprintf("./tmp/test/%d/", id), "friend")
	cfg.Friend.MinSyncRandomSeconds = 5
	cfg.Friend.MaxSyncRandomSeconds = 7
	fmt.Printf("me config: %v\n", cfg.Me)

	err = cfg.Me.SetMyKey("", "", "", false)
	if err != nil {
		return nil, err
	}

	n, err := node.New(cfg.Node)
	if err != nil {
		return nil, err
	}

	n.Register(func(ctx *service.RouterContext) (service.NodeRouter, error) {
		nodeKey := cfg.Node.NodeKey()
		nodeID := discover.PubkeyID(&nodeKey.PublicKey)

		ptt, err := service.NewRouter(ctx, cfg.Router, &nodeID, nodeKey)
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

	return n, nil
}
