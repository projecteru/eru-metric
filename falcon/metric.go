package falcon

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/HunanTV/eru-agent/logs"
	"github.com/fsouza/go-dockerclient"
	"github.com/open-falcon/common/model"
)

func (self *Metric) InitMetric(client DockerClient, cid string, pid int) (err error) {
	if self.statFile, err = os.Open(fmt.Sprintf("/proc/%d/net/dev", pid)); err != nil {
		return
	}
	if info, err := self.UpdateStats(client, cid); err == nil {
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

func (self *Metric) UpdateStats(client DockerClient, cid string) (map[string]uint64, error) {
	info := map[string]uint64{}
	statsChan := make(chan *docker.Stats)
	doneChan := make(chan bool)
	opt := docker.StatsOptions{cid, statsChan, false, doneChan, time.Duration(STATS_TIMEOUT * time.Second)}
	go func() {
		if err := client.Stats(opt); err != nil {
			logs.Info("Get stats failed", cid[:12], err)
		}
	}()

	stats := &docker.Stats{}
	select {
	case stats = <-statsChan:
		if stats == nil {
			return info, errors.New("Get stats failed")
		}
	case <-time.After(STATS_FORCE_DONE * time.Second):
		doneChan <- true
		return info, errors.New("Get stats timeout")
	}

	info["cpu_user"] = stats.CPUStats.CPUUsage.UsageInUsermode
	info["cpu_system"] = stats.CPUStats.CPUUsage.UsageInKernelmode
	info["cpu_usage"] = stats.CPUStats.CPUUsage.TotalUsage
	//FIXME in container it will get all CPUStats
	info["mem_usage"] = stats.MemoryStats.Usage
	info["mem_max_usage"] = stats.MemoryStats.MaxUsage
	info["mem_rss"] = stats.MemoryStats.Stats.Rss

	if err := self.GetNetStats(info); err != nil {
		return info, err
	}
	return info, nil
}

func (self *Metric) SaveLast(info map[string]uint64) {
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
		case (strings.HasPrefix(k, VLAN_PREFIX) || strings.HasPrefix(k, DEFAULT_BR)) && d >= self.Save[k]:
			rate[fmt.Sprintf("%s.rate", k)] = float64(d-self.Save[k]) / second_t
		case strings.HasPrefix(k, "mem"):
			rate[k] = float64(d)
		}
	}
	self.Last = now
	return
}

func (self *Metric) Send(rate map[string]float64) error {
	data := []*model.MetricValue{}
	for k, d := range rate {
		data = append(data, self.newMetricValue(k, d))
	}
	var resp model.TransferResponse
	if err := self.Client.Call("Transfer.Update", data, &resp); err != nil {
		return err
	}
	logs.Debug(data)
	logs.Debug(self.Endpoint, self.Last, &resp)
	return nil
}

func (self *Metric) newMetricValue(metric string, value interface{}) *model.MetricValue {
	mv := &model.MetricValue{
		Endpoint:  self.Endpoint,
		Metric:    metric,
		Value:     value,
		Step:      int64(self.Step.Seconds()),
		Type:      "GAUGE",
		Tags:      self.Tag,
		Timestamp: self.Last.Unix(),
	}
	return mv
}
