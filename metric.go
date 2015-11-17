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

func (self *Metric) InitMetric(dockerclient *docker.Client, cid string, pid int) error {
	if self.statFile, err = os.Open(fmt.Sprintf("/proc/%d/net/dev", pid)); err != nil {
		return err
	}
	info, upOk := self.UpdateStats(cid, dockerclient)
	if !upOk {
		return errors.New("Init metric failed")
	}
	self.Last = time.Now()
	self.saveLast(info)
	return nil
}

func (self *Metric) Exit() {
	defer self.statFile.Close()
	self.Stop <- true
	close(self.Stop)
}

func (self *Metric) UpdateStats(dockerclient *docker.Client, cid string) (map[string]uint64, bool) {
	info := map[string]uint64{}
	statsChan := make(chan *docker.Stats)
	doneChan := make(chan bool)
	opt := docker.StatsOptions{cid, statsChan, false, doneChan, time.Duration(STATS_TIMEOUT * time.Second)}
	go func() {
		if err := dockerclient.Stats(opt); err != nil {
			logs.Info("Get stats failed", cid[:12], err)
		}
	}()

	select {
	case stats := <-statsChan:
		if stats == nil {
			return info, false
		}
	case <-time.After(STATS_FORCE_DONE * time.Second):
		doneChan <- true
		return info, false
	}

	info["cpu_user"] = stats.CPUStats.CPUUsage.UsageInUsermode
	info["cpu_system"] = stats.CPUStats.CPUUsage.UsageInKernelmode
	info["cpu_usage"] = stats.CPUStats.CPUUsage.TotalUsage
	//FIXME in container it will get all CPUStats
	info["mem_usage"] = stats.MemoryStats.Usage
	info["mem_max_usage"] = stats.MemoryStats.MaxUsage
	info["mem_rss"] = stats.MemoryStats.Stats.Rss

	if err := GetNetStats(self.statFile, info); err != nil {
		logs.Info("Get net stats failed", cid[:12], err)
		return info, false
	}
	return info, true
}

func (self *Metrics) SaveLast(info map[string]uint64) {
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
}

func (self *EruApp) newMetricValue(metric string, value interface{}) *model.MetricValue {
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
