Eru-Metric
==========

[![GoDoc](https://godoc.org/github.com/HunanTV/eru-metric?status.svg)](https://godoc.org/github.com/HunanTV/eru-metric)

A library for watching container metrics and send to remote.

This repo implement open-falcon methods you can write your methods by your self.

How
===

* write a func to implement Send method if you want send metircs to other place.

```
func Send(data map[string]float64, endpoint, tag string, timestamp, step int64) error
```

* set metric global setting

```
SetGlobalSetting(client Remote, timeout, forceTimeout time.Duration, vlanPrefix, defaultVlan string)
```

* create a backend object which implemented Send interface.

* create a metric for each container.

```
CreateMetric(step time.Duration, client Remote, tag, endpoint string)
```

* init metric object

```
InitMetric(cid string, pid int)
```

* update, calcuate, save and send

```
UpdateStats(cid string)
CalcRate(info map[string]uint64, now time.Time)
SaveLast(info map[string]uint64)
Send(rate map[string]float64)
```

* exit metirc

```
Exit()
```

Example
=======

see example.go only work under LINUX environment.

```
eru-metric CONTAINERID CONTAINERID ... CONTAINERID [-DEBUG] [-d docker remote addr] [-t transfer remote addr]
```

