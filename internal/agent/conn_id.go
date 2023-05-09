package agent

import "sync/atomic"

var connId uint32

func genConnId() uint32 {
	return atomic.AddUint32(&connId, 1)
}
