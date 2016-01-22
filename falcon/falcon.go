package falcon

import (
	"math"
	"net/rpc"
	"sync"
	"time"

	"github.com/projecteru/eru-agent/logs"
	"github.com/open-falcon/common/model"
	"github.com/toolkits/net"
)

func CreateFalconClient(transfer string, timeout time.Duration) *FalconClient {
	return &FalconClient{
		RpcServer: transfer,
		Timeout:   timeout,
	}
}

type FalconClient struct {
	sync.Mutex
	rpcClient *rpc.Client
	RpcServer string
	Timeout   time.Duration
}

func (self *FalconClient) Close() error {
	if self.rpcClient != nil {
		self.rpcClient.Close()
		self.rpcClient = nil
	}
	return nil
}

func (self *FalconClient) insureConn() error {
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

func (self *FalconClient) call(method string, args interface{}, reply interface{}) error {
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

func (self *FalconClient) Send(data map[string]float64, endpoint, tag string, timestamp, step int64) error {
	metrics := []*model.MetricValue{}
	var metric *model.MetricValue
	for k, v := range data {
		metric = &model.MetricValue{
			Endpoint:  endpoint,
			Metric:    k,
			Value:     v,
			Step:      step,
			Type:      "GAUGE",
			Tags:      tag,
			Timestamp: timestamp,
		}
		metrics = append(metrics, metric)
	}
	logs.Debug(metrics)
	var resp model.TransferResponse
	if err := self.call("Transfer.Update", metrics, &resp); err != nil {
		return err
	}
	logs.Debug(endpoint, timestamp, &resp)
	return nil
}
