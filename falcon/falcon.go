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
