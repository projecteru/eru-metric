package statsd

import (
	"fmt"

	statsdlib "github.com/CMGS/statsd"
	"github.com/projecteru/eru-agent/logs"
)

func CreateStatsDClient(addr string) *StatsDClient {
	return &StatsDClient{
		Addr: addr,
	}
}

type StatsDClient struct {
	Addr string
}

func (self *StatsDClient) Close() error {
	return nil
}

func (self *StatsDClient) Send(data map[string]float64, endpoint, tag string, timestamp, step int64) error {
	remote, err := statsdlib.New("addr")
	if err != nil {
		logs.Info("Connect statsd failed")
		return err
	}
	defer remote.Close()
	defer remote.Flush()
	for k, v := range data {
		key := fmt.Sprintf("%s.%s.%s", endpoint, tag, k)
		remote.Gauge(key, v)
	}
	return nil
}
