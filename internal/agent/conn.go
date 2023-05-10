package agent

import (
	"bytes"
	"encoding/binary"
	"github.com/Orlion/hersql/internal/mysql"
	"net"
)

var DEFAULT_CAPABILITY uint32 = mysql.CLIENT_LONG_PASSWORD | mysql.CLIENT_LONG_FLAG |
	mysql.CLIENT_CONNECT_WITH_DB | mysql.CLIENT_PROTOCOL_41 |
	mysql.CLIENT_TRANSACTIONS | mysql.CLIENT_SECURE_CONNECTION | mysql.CLIENT_PLUGIN_AUTH

type conn struct {
	pkg        *mysql.PacketIO
	connId     uint32
	salt       []byte
	agent      *Agent
	rwc        net.Conn
	remoteAddr string
	status     uint16
	capability uint32
	user       string
	db         string
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

		data, err := c.readPacket()
		if err != nil {
			// todo: log
			break
		}

		// 发送到服务端
		println(data)
	}
}

func (c *conn) handshake() error {
	if err := c.writeInitialHandshake(); err != nil {
		return err
	}

	if err := c.readHandshakeResponse(); err != nil {
		return err
	}

	if err := c.writeOK(nil); err != nil {
		return err
	}

	c.pkg.Sequence = 0

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

func (c *conn) readHandshakeResponse() error {
	data, err := c.readPacket()

	if err != nil {
		return err
	}

	pos := 0

	//capability
	c.capability = binary.LittleEndian.Uint32(data[:4])
	pos += 4

	//skip max packet size
	pos += 4

	//charset, skip, if you want to use another charset, use set names
	//c.collation = CollationId(data[pos])
	pos++

	//skip reserved 23[00]
	pos += 23

	//user name
	c.user = string(data[pos : pos+bytes.IndexByte(data[pos:], 0)])
	pos += len(c.user) + 1

	//auth length and auth
	authLen := int(data[pos])
	pos++

	// skip auth

	pos += authLen

	if c.capability&mysql.CLIENT_CONNECT_WITH_DB > 0 {
		if len(data[pos:]) == 0 {
			return nil
		}

		db := string(data[pos : pos+bytes.IndexByte(data[pos:], 0)])
		pos += len(c.db) + 1
		c.db = db
	}

	return nil
}

func (c *conn) writeOK(r *mysql.Result) error {
	if r == nil {
		r = &mysql.Result{Status: c.status}
	}
	data := make([]byte, 4, 32)

	data = append(data, mysql.OK_HEADER)

	data = append(data, mysql.PutLengthEncodedInt(r.AffectedRows)...)
	data = append(data, mysql.PutLengthEncodedInt(r.InsertId)...)

	if c.capability&mysql.CLIENT_PROTOCOL_41 > 0 {
		data = append(data, byte(r.Status), byte(r.Status>>8))
		data = append(data, 0, 0)
	}

	return c.writePacket(data)
}

func (c *conn) writePacket(data []byte) error {
	return c.pkg.WritePacket(data)
}

func (c *conn) readPacket() ([]byte, error) {
	return c.pkg.ReadPacket()
}

func (c *conn) close() error {
	err := c.rwc.Close()
	return err
}
