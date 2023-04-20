package agent

import (
	"net"
)

type conn struct {
	agent      *Agent
	rwc        net.Conn
	remoteAddr string
}

func (c *conn) serve() {
	defer func() {
		c.close()
		c.agent.decrConnNum()
		if err := recover(); err != nil {
			// todo: log
		}
	}()

	c.remoteAddr = c.rwc.RemoteAddr().String()

	err := c.handshake()
	if err != nil {
		// todo: log
		return
	}

	for {
		if c.agent.shuttingDown() {
			break
		}

		p, err := c.readPacket()
		if err != nil {
			// todo: log
			break
		}

		// 发送到服务端
		println(p)
	}
}

func (c *conn) handshake() (err error) {

	return
}

func (c *conn) close() error {
	err := c.rwc.Close()
	return err
}
