package agent

import (
	"github.com/Orlion/hersql/pkg/atomicx"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Agent struct {
	inShutdown atomicx.Bool
	mu         sync.Mutex
	listener   net.Listener
	doneChan   chan struct{}
	connNum    int64
}

func NewAgent() *Agent {
	return new(Agent)
}

func (agent *Agent) ListenAndServe() (err error) {
	agent.listener, err = net.Listen("tcp", ":3306")
	if err != nil {
		return
	}

	go agent.serve()
	return
}

func (agent *Agent) serve() {
	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		rw, err := agent.listener.Accept()
		if err != nil {
			select {
			case <-agent.getDoneChan():
				return
			default:
			}

			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}

				time.Sleep(tempDelay)
				continue
			}

			return
		}

		c := agent.newConn(rw)
		go func() {
			c.serve()
		}()
	}
}

func (agent *Agent) shuttingDown() bool {
	return agent.inShutdown.Get()
}

func (agent *Agent) newConn(rwc net.Conn) *conn {
	c := &conn{
		agent: agent,
		rwc:   rwc,
	}

	agent.incrConnNum()

	return c
}

func (agent *Agent) getDoneChan() <-chan struct{} {
	agent.mu.Lock()
	defer agent.mu.Unlock()
	return agent.getDoneChanLocked()
}

func (agent *Agent) getDoneChanLocked() chan struct{} {
	if agent.doneChan == nil {
		agent.doneChan = make(chan struct{})
	}
	return agent.doneChan
}

func (agent *Agent) getConnNum() int64 {
	return atomic.LoadInt64(&agent.connNum)
}

func (agent *Agent) incrConnNum() {
	atomic.AddInt64(&agent.connNum, 1)
}

func (agent *Agent) decrConnNum() {
	atomic.AddInt64(&agent.connNum, -1)
}
