package metric

import (
	"os"
	"sync"
	"time"

	"github.com/fsouza/go-dockerclient"
)

type DockerClient interface {
	Stats(opts docker.StatsOptions) error
}

type Remote interface {
	Send(data map[string]float64, endpoint, tag string, timestamp, step int64) error
	Close() error
}

type Metric struct {
	*sync.Mutex
	Step     time.Duration
	Client   Remote
	Tag      string
	Endpoint string

	statFile *os.File
	Last     time.Time

	Stop chan bool
	Save map[string]uint64
}

type Setting struct {
	timeout     time.Duration
	force       time.Duration
	vlanPrefix  string
	defaultVlan string
	client      DockerClient
}

var g Setting
