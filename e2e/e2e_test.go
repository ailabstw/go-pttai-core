package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/ailabstw/go-pttai-core/account"
	"github.com/ailabstw/go-pttai-core/friend"
	"github.com/ailabstw/go-pttai-core/log"
	"github.com/ailabstw/go-pttai-core/me"
	"github.com/ailabstw/go-pttai-core/node"
	"github.com/ailabstw/go-pttai-core/p2p/discover"
	"github.com/ailabstw/go-pttai-core/service"
	signalserver "github.com/ailabstw/pttai-signal-server"
	"github.com/gorilla/mux"
	baloo "gopkg.in/h2non/baloo.v3"
)

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

func init() {
	// log.Root().SetHandler(log.Must.FileHandler("./tmp/test/log.tmp.txt", log.TerminalFormat(false)))
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
	cfg.Me.DataDir = filepath.Join(cfg.Node.DataDir, "me")
	cfg.Router.DataDir = filepath.Join(fmt.Sprintf("./tmp/test/%d/", id), "service")
	cfg.Account.DataDir = filepath.Join(fmt.Sprintf("./tmp/test/%d/", id), "account")
	cfg.Friend.DataDir = filepath.Join(fmt.Sprintf("./tmp/test/%d/", id), "friend")
	fmt.Printf("me config: %v\n", cfg.Me)

	err := cfg.Me.SetMyKey("", "", "", false)
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

// helpers

func testCore(c *baloo.Client, bodyString string, data interface{}, t *testing.T, isDebug bool) ([]byte, *myError) {
	rbody := &rBody{}

	c.Post("/").
		BodyString(bodyString).
		SetHeader("Content-Type", "application/json").
		Expect(t).
		AssertFunc(getResponseBody(rbody, t)).
		Done()

	var wrapper *dataWrapper
	err := &myError{}
	if data != nil {
		wrapper = &dataWrapper{Result: data, Error: err}
		err := json.Unmarshal(rbody.Body, wrapper)
		if err != nil {
			t.Logf("unable to parse: b: %v e: %v", rbody.Body, err)
		}
	}

	if isDebug {
		if data != nil {
			t.Logf("after Parse: body: %v data: %v", string(rbody.Body), wrapper.Result)
		} else {
			t.Logf("after Parse: body: %v", string(rbody.Body))

		}
	}

	return rbody.Body, err
}

type rBody struct {
	Header        map[string][]string
	Body          []byte
	ContentLength int64
}

type dataWrapper struct {
	Result interface{}
	Error  *myError
}

type myError struct {
	Code int
	Msg  string
}

func getResponseBody(r *rBody, t *testing.T) func(res *http.Response, req *http.Request) error {
	return func(res *http.Response, req *http.Request) error {
		body, err := readBody(res, t)
		if err != nil {
			return err
		}
		r.Body = body
		r.ContentLength = res.ContentLength
		r.Header = res.Header
		return nil
	}
}

func readBody(res *http.Response, t *testing.T) ([]byte, error) {
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Logf("[ERROR] Unable to read body: e: %v", err)
		return []byte{}, err
	}
	// Re-fill body reader stream after reading it
	res.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return body, err
}

func testListCore(c *baloo.Client, bodyString string, data interface{}, t *testing.T, isDebug bool) []byte {
	rbody := &rBody{}

	c.Post("/").
		BodyString(bodyString).
		SetHeader("Content-Type", "application/json").
		Expect(t).
		AssertFunc(getResponseBody(rbody, t)).
		Done()

	ParseBody(rbody.Body, t, data, true)

	if isDebug {
		t.Logf("after Parse: length: %v header: %v body: %v data: %v", rbody.ContentLength, rbody.Header, string(rbody.Body), data)
	}

	return rbody.Body
}

func ParseBody(b []byte, t *testing.T, data interface{}, isList bool) {
	err := json.Unmarshal(b, data)
	if err != nil && !isList {
		t.Logf("unable to parse: b: %v e: %v", b, err)
	}
}

func startSignalServer() {
	addr := "127.0.0.1:9489"
	go func() {
		server := signalserver.NewServer()

		srv := &http.Server{Addr: addr}
		r := mux.NewRouter()
		r.HandleFunc("/signal", server.SignalHandler)
		srv.Handler = r

		srv.ListenAndServe()
	}()
}

func runNode(ctx context.Context, id int, port int) error {
	c := exec.CommandContext(
		ctx,
		"./bin/bin",
		fmt.Sprintf("%d", id),
		fmt.Sprintf("%d", port),
	)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Start()

}
