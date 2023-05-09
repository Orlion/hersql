package agent

import (
	"github.com/Orlion/hersql/internal/mysql"
	"net"
)

var DEFAULT_CAPABILITY uint32 = CLIENT_LONG_PASSWORD | CLIENT_LONG_FLAG |
	CLIENT_CONNECT_WITH_DB | CLIENT_PROTOCOL_41 |
	CLIENT_TRANSACTIONS | CLIENT_SECURE_CONNECTION

type conn struct {
	connId     uint32
	salt       []byte
	agent      *Agent
	rwc        net.Conn
	remoteAddr string
	status     uint16
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

func (c *conn) handshake() error {

	return nil
}

func (c *conn) writeInitialHandshake() error {
	// 服务端给客户端发一个初始化握手包
	data := make([]byte, 4, 128)

	//protocol version Always 10
	data = append(data, 10)

	//server version[00]
	data = append(data, mysql.ServerVersion...)
	data = append(data, 0)

	// thread id
	data = append(data, byte(c.connId), byte(c.connId>>8), byte(c.connId>>16), byte(c.connId>>24))

	//auth-plugin-data-part-1
	data = append(data, c.salt[0:8]...)

	//filter [00]
	data = append(data, 0)

	//capability flag lower 2 bytes, using default capability here
	data = append(data, byte(DEFAULT_CAPABILITY), byte(DEFAULT_CAPABILITY>>8))

	//charset, utf-8 default
	data = append(data, uint8(mysql.DEFAULT_COLLATION_ID))

	//status
	data = append(data, byte(c.status), byte(c.status>>8))

	//below 13 byte may not be used
	//capability flag upper 2 bytes, using default capability here
	data = append(data, byte(DEFAULT_CAPABILITY>>16), byte(DEFAULT_CAPABILITY>>24))

	//filter [0x15], for wireshark dump, value is 0x15
	data = append(data, 0x15)

	//reserved 10 [00]
	data = append(data, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0)

	//auth-plugin-data-part-2
	data = append(data, c.salt[8:]...)

	//filter [00]
	data = append(data, 0)

	return c.writePacket(data)
}

func (c *conn) writePacket(data []byte) error {
	return nil
}

func (c *conn) close() error {
	err := c.rwc.Close()
	return err
}
