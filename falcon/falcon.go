package falcon

import (
	"time"
)

func CreateRPCClient(transfer string, timeout time.Duration) SingleConnRpcClient {
	return SingleConnRpcClient{
		RpcServer: transfer,
		Timeout:   timeout,
	}
}

func CreateMetric(step time.Duration, client SingleConnRpcClient, tag string, endpoint string) Metric {
	return Metric{
		Step:     step,
		Client:   client,
		Tag:      tag,
		Endpoint: endpoint,
	}
}
