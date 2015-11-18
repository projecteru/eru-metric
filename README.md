Eru-Metric
==========

A library for watching container metrics and send to remote.

This repo implement open-falcon methods you can write your methods by your self.

How
===

Just write a func to implement this interface

```
func Send(data map[string]float64, endpoint, tag string, timestamp, step int64) error
```

