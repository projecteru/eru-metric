package falcon

import (
	"time"

	"github.com/fsouza/go-dockerclient"
)

func main() {
	dockerclient := docker.NewClient("http://localhost:12345")
	client := CreateRPCClient("127.0.0.1:5000", time.Duration(5))
	metric := CreateMetric(time.Duration(30)*time.Second, client, "a=b,b=c", "test_endpoint")

	// Get container pid from docker inspect
	pid := 12345
	cid := "testcontainerid"

	if err := metric.InitMetric(dockerclient, cid, pid); err != nil {
		// init failed
		return
	}

	for {
		select {
		case now := <-time.Tick(metric.Step):
			go func() {
				info, upOk := metric.UpdateStats(dockerclient, cid)
				if !upOk {
					return
				}
				rate := metric.CalcRate(info, now)
				metric.SaveLast(info)
				// for safe
				go metric.Send(rate)
			}()
		case <-metric.Stop:
			return
		}
	}
}
