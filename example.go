package main

import (
	"fmt"
	"time"

	"github.com/HunanTV/eru-agent/logs"
	"github.com/HunanTV/eru-metric/falcon"
	"github.com/HunanTV/eru-metric/metric"
	"github.com/fsouza/go-dockerclient"
)

func main() {
	logs.Mode = true
	metric.SetMetricConfig(2, 3, "vnbe", "eth0")
	cert := "/Users/CMGS/.docker/machine/machines/default/cert.pem"
	key := "/Users/CMGS/.docker/machine/machines/default/key.pem"
	ca := "/Users/CMGS/.docker/machine/machines/default/ca.pem"
	dockerclient, err := docker.NewTLSClient("tcp://192.168.99.100:2376", cert, key, ca)
	if err != nil {
		fmt.Println(err)
		return
	}
	client := falcon.CreateRPCClient("10.200.8.37:8433", time.Duration(5))
	serv := metric.CreateMetric(time.Duration(5)*time.Second, client, "a=b,b=c", "test_endpoint")

	// Get container pid from docker inspect
	pid := 5936
	cid := "17370fa463b5"

	if err := serv.InitMetric(dockerclient, cid, pid); err != nil {
		// init failed
		fmt.Println("failed", err)
		return
	}

	println("begin")
	for {
		select {
		case now := <-time.Tick(serv.Step):
			go func() {
				if info, err := serv.UpdateStats(dockerclient, cid); err == nil {
					fmt.Println(info)
					rate := serv.CalcRate(info, now)
					serv.SaveLast(info)
					// for safe
					fmt.Println(rate)
					go serv.Send(rate)
				}
			}()
		case <-serv.Stop:
			return
		}
	}
}
