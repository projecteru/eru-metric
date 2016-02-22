package statsd

import (
	"fmt"
	"sync"

	statsdlib "github.com/cactus/go-statsd-client/statsd"
)

func CreateStatsDClient(addr string) *StatsDClient {
	return &StatsDClient{
		Addr: addr,
	}
}

type StatsDClient struct {
	sync.Mutex
	Addr string
}

func (self *StatsDClient) Close() error {
	return nil
}

func (self *StatsDClient) Send(data map[string]float64, endpoint, tag string, timestamp, step int64) error {
	remote, err := statsdlib.NewClient(self.Addr, "")
	defer remote.Close()
	if err != nil {
		return err
	}
	for k, v := range data {
		key := fmt.Sprintf("%s.%s.%s", endpoint, tag, k)
		remote.Raw(key, fmt.Sprintf("%v", v), 1.0)
	}
	return nil
}
