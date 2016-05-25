package main

import (
	"flag"
	"fmt"
	"time"

	"golang.org/x/net/context"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/engine-api/client"
	"github.com/projecteru/eru-metric/metric"
	"github.com/projecteru/eru-metric/statsd"
)

func main() {
	var dockerAddr string
	var transferAddr string
	var debug bool
	flag.BoolVar(&debug, "DEBUG", false, "enable debug")
	flag.StringVar(&dockerAddr, "d", "tcp://192.168.99.100:2376", "docker daemon addr")
	flag.StringVar(&transferAddr, "t", "10.200.8.37:8433", "transfer addr")
	flag.Parse()
	if flag.NArg() < 1 {
		fmt.Println("need at least one container id")
		return
	}
	if debug {
		log.SetLevel(log.DebugLevel)
	}

	cli, _ := client.NewEnvClient()

	metric.SetGlobalSetting(cli, 2, 3, "vnbe", "eth0")
	client := statsd.CreateStatsDClient(transferAddr)
	ctx := context.Background()

	for i := 0; i < flag.NArg(); i++ {
		if c, err := cli.ContainerInspect(ctx, flag.Arg(i)); err != nil {
			fmt.Println(flag.Arg(i), err)
			continue
		} else {
			go start_watcher(client, c.ID, c.State.Pid)
		}
	}
	for {
	}
}

func start_watcher(client metric.Remote, cid string, pid int) {
	serv := metric.CreateMetric(time.Duration(5)*time.Second, client, "a.b", fmt.Sprintf("test_%s", cid[:7]))
	if err := serv.InitMetric(cid, pid); err != nil {
		fmt.Println("failed", err)
		return
	}

	t := time.NewTicker(serv.Step)
	defer t.Stop()
	fmt.Println("begin watch", cid)
	for {
		select {
		case now := <-t.C:
			go func() {
				if info, err := serv.UpdateStats(cid); err == nil {
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
