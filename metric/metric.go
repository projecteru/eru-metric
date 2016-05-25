package metric

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/docker/engine-api/types"
	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
)

func SetGlobalSetting(client DockerClient, timeout, force time.Duration, vlanPrefix, defaultVlan string) {
	g = Setting{timeout, force, vlanPrefix, defaultVlan, client}
}

func CreateMetric(step time.Duration, client Remote, tag string, endpoint string) Metric {
	return Metric{
		Step:     step,
		Client:   client,
		Tag:      tag,
		Endpoint: endpoint,
		Stop:     make(chan bool),
	}
}

func (self *Metric) InitMetric(cid string, pid int) (err error) {
	if self.statFile, err = os.Open(fmt.Sprintf("/proc/%d/net/dev", pid)); err != nil {
		return
	}
	var info map[string]uint64
	if info, err = self.UpdateStats(cid); err == nil {
		self.Last = time.Now()
		self.SaveLast(info)
	}
	return
}

func (self *Metric) Exit() {
	defer self.statFile.Close()
	self.Stop <- true
	close(self.Stop)
}

func (self *Metric) UpdateStats(cid string) (map[string]uint64, error) {
	info := map[string]uint64{}
	ctx := context.Background()
	resp, err := g.client.ContainerStats(ctx, cid, false)
	if err != nil {
		log.Errorf("Get stats failed %s %s", cid[:12], err)
		return info, err
	}
	defer resp.Close()
	data, err := ioutil.ReadAll(resp)
	if err != nil {
		log.Errorf("Read stats failed %s %s", cid[:12], err)
		return info, err
	}
	var stats types.StatsJSON
	if err := json.Unmarshal(data, &stats); err != nil {
		log.Errorf("Unmarshal stats failed %s %s", cid[:12], err)
		return info, err
	}

	info["cpu_user"] = stats.Stats.CPUStats.CPUUsage.UsageInUsermode
	info["cpu_system"] = stats.Stats.CPUStats.CPUUsage.UsageInKernelmode
	info["cpu_usage"] = stats.Stats.CPUStats.CPUUsage.TotalUsage
	//FIXME in container it will get all CPUStats
	info["mem_usage"] = stats.Stats.MemoryStats.Usage
	info["mem_max_usage"] = stats.Stats.MemoryStats.MaxUsage
	info["mem_rss"] = stats.Stats.MemoryStats.Stats["rss"]

	if err := self.getNetStats(info); err != nil {
		return info, err
	}
	return info, nil
}

func (self *Metric) SaveLast(info map[string]uint64) {
	self.Lock()
	defer self.Unlock()
	self.Save = map[string]uint64{}
	for k, d := range info {
		self.Save[k] = d
	}
}

func (self *Metric) CalcRate(info map[string]uint64, now time.Time) (rate map[string]float64) {
	rate = map[string]float64{}
	delta := now.Sub(self.Last)
	nano_t := float64(delta.Nanoseconds())
	second_t := delta.Seconds()
	for k, d := range info {
		switch {
		case strings.HasPrefix(k, "cpu_") && d >= self.Save[k]:
			rate[fmt.Sprintf("%s_rate", k)] = float64(d-self.Save[k]) / nano_t
		case (strings.HasPrefix(k, g.vlanPrefix) || strings.HasPrefix(k, g.defaultVlan)) && d >= self.Save[k]:
			rate[fmt.Sprintf("%s.rate", k)] = float64(d-self.Save[k]) / second_t
		case strings.HasPrefix(k, "mem"):
			rate[k] = float64(d)
		}
	}
	self.Last = now
	return
}

func (self *Metric) Send(rate map[string]float64) error {
	step := int64(self.Step.Seconds())
	timestamp := self.Last.Unix()
	return self.Client.Send(rate, self.Endpoint, self.Tag, timestamp, step)
}
