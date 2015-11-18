package falcon

import (
	"math"
	"net/rpc"
	"os"
	"sync"
	"time"

	"github.com/HunanTV/eru-agent/logs"
	"github.com/fsouza/go-dockerclient"
	"github.com/toolkits/net"
)

const (
	STATS_TIMEOUT    = 2
	STATS_FORCE_DONE = 3

	VLAN_PREFIX = "vnbe"
	DEFAULT_BR  = "eth0"
)

type DockerClient interface {
	Stats(opts docker.StatsOptions) error
}

type Metric struct {
	Step     time.Duration
	Client   SingleConnRpcClient
	Tag      string
	Endpoint string

	statFile *os.File
	Last     time.Time

	Stop chan bool
	Save map[string]uint64
}

type SingleConnRpcClient struct {
	sync.Mutex
	rpcClient *rpc.Client
	RpcServer string
	Timeout   time.Duration
}

func (self *SingleConnRpcClient) Close() {
	if self.rpcClient != nil {
		self.rpcClient.Close()
		self.rpcClient = nil
	}
}

func (self *SingleConnRpcClient) insureConn() error {
	if self.rpcClient != nil {
		return nil
	}

	var err error
	var retry int = 1

	for {
		if self.rpcClient != nil {
			return nil
		}

		self.rpcClient, err = net.JsonRpcClient("tcp", self.RpcServer, self.Timeout)
		if err == nil {
			return nil
		}

		logs.Info("Metrics rpc dial fail", err)
		if retry > 5 {
			return err
		}

		time.Sleep(time.Duration(math.Pow(2.0, float64(retry))) * time.Second)
		retry++
	}
	return nil
}

func (self *SingleConnRpcClient) Call(method string, args interface{}, reply interface{}) error {

	self.Lock()
	defer self.Unlock()

	if err := self.insureConn(); err != nil {
		return err
	}

	timeout := time.Duration(50 * time.Second)
	done := make(chan error)

	go func() {
		err := self.rpcClient.Call(method, args, reply)
		done <- err
	}()

	select {
	case <-time.After(timeout):
		logs.Info("Metrics rpc call timeout", self.rpcClient, self.RpcServer)
		self.Close()
	case err := <-done:
		if err != nil {
			self.Close()
			return err
		}
	}

	return nil
}
