package main

import (
	"fmt"
	"time"

	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-falcon/falcon"
	"github.com/fsouza/go-dockerclient"
)

func main() {
	logs.Mode = true
	cert := "/Users/CMGS/.docker/machine/machines/default/cert.pem"
	key := "/Users/CMGS/.docker/machine/machines/default/key.pem"
	ca := "/Users/CMGS/.docker/machine/machines/default/ca.pem"
	dockerclient, _ := docker.NewTLSClient("tcp://192.168.99.100:2376", cert, key, ca)
	client := falcon.CreateRPCClient("10.200.8.37:8433", time.Duration(5))
	metric := falcon.CreateMetric(time.Duration(30)*time.Second, client, "a=b,b=c", "test_endpoint")

	// Get container pid from docker inspect
	pid := 4330
	cid := "eb622e78de3f"

	if err := metric.InitMetric(dockerclient, cid, pid); err != nil {
		// init failed
		fmt.Println("failed", err)
		return
	}

	println("begin")
	for {
		select {
		case now := <-time.Tick(metric.Step):
			go func() {
				if info, err := metric.UpdateStats(dockerclient, cid); err == nil {
					fmt.Println(info)
					rate := metric.CalcRate(info, now)
					metric.SaveLast(info)
					// for safe
					fmt.Println(rate)
					go metric.Send(rate)
				}
			}()
		case <-metric.Stop:
			return
		}
	}
}
