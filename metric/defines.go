package metric

import (
	"os"
	"time"

	"github.com/fsouza/go-dockerclient"
)

var STATS_TIMEOUT time.Duration = 2
var STATS_FORCE_DONE time.Duration = 3

var VLAN_PREFIX string = "vnbe"
var DEFAULT_BR string = "eth0"

type DockerClient interface {
	Stats(opts docker.StatsOptions) error
}

type Remote interface {
	Send(data map[string]float64, endpoint, tag string, timestamp, step int64) error
}

type Metric struct {
	Step     time.Duration
	Client   Remote
	Tag      string
	Endpoint string

	statFile *os.File
	Last     time.Time

	Stop chan bool
	Save map[string]uint64
}
