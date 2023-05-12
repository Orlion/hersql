package entrance

import (
	"bytes"
	"encoding/binary"
	"errors"
	"github.com/Orlion/hersql/log"
	mysql2 "github.com/Orlion/hersql/mysql"
	"io"
	"net"
)

var DEFAULT_CAPABILITY uint32 = mysql2.CLIENT_LONG_PASSWORD | mysql2.CLIENT_LONG_FLAG |
	mysql2.CLIENT_CONNECT_WITH_DB | mysql2.CLIENT_PROTOCOL_41 |
	mysql2.CLIENT_TRANSACTIONS | mysql2.CLIENT_SECURE_CONNECTION | mysql2.CLIENT_PLUGIN_AUTH

type Conn struct {
	connId     uint32
	server     *Server
	rwc        net.Conn
	remoteAddr string
	pkg        *mysql2.PacketIO
	salt       []byte
	status     uint16
	capability uint32
	user       string
	db         string
	password   string
	exitConnId uint64
}

func (c *Conn) serve() {
	defer func() {
		c.close()
		c.server.decrConnNum()
		if err := recover(); err != nil {
			log.Errorf("Conn serve panic, err: %v", err)
		}
	}()

	c.remoteAddr = c.rwc.RemoteAddr().String()

	err := c.handshake()
	if err != nil {
		// todo: log
		return
	}

	for {
		if c.server.shuttingDown() {
			break
		}

		data, err := c.readPacket()
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Infof("Conn from %s closed", c.remoteAddr)
			} else {
				log.Errorf("Conn read packet from %s error: %s", c.remoteAddr, err.Error())
			}
			break
		}

		// 发送到服务端
		if responseData, err := c.exitTransport(data); err != nil {
			log.Errorf("transport error: %s", err.Error())
		} else {
			if err = c.writePacket(responseData); err != nil {
				log.Errorf("write packet to %s error: %s", c.remoteAddr, err.Error())
			}
		}
	}
}

func (c *Conn) handshake() error {
	if err := c.writeInitialHandshake(); err != nil {
		return err
	}

	if err := c.readHandshakeResponse(); err != nil {
		return err
	}

	if err := c.exitConnect(); err != nil {
		return err
	}

	if err := c.writeOK(nil); err != nil {
		return err
	}

	c.pkg.Sequence = 0

	return nil
}

func (c *Conn) writeInitialHandshake() error {
	data := make([]byte, 4, 128)

	//protocol version Always 10
	data = append(data, 10)

	//exit version[00]
	data = append(data, mysql2.ServerVersion...)
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
	data = append(data, uint8(mysql2.DEFAULT_COLLATION_ID))

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

func (c *Conn) readHandshakeResponse() error {
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

	_ = string(data[pos : pos+authLen]) // todo

	pos += authLen

	if c.capability&mysql2.CLIENT_CONNECT_WITH_DB > 0 {
		if len(data[pos:]) == 0 {
			return nil
		}

		db := string(data[pos : pos+bytes.IndexByte(data[pos:], 0)])
		pos += len(c.db) + 1
		c.db = db
	}

	return nil
}

func (c *Conn) writeOK(r *mysql2.Result) error {
	if r == nil {
		r = &mysql2.Result{Status: c.status}
	}
	data := make([]byte, 4, 32)

	data = append(data, mysql2.OK_HEADER)

	data = append(data, mysql2.PutLengthEncodedInt(r.AffectedRows)...)
	data = append(data, mysql2.PutLengthEncodedInt(r.InsertId)...)

	if c.capability&mysql2.CLIENT_PROTOCOL_41 > 0 {
		data = append(data, byte(r.Status), byte(r.Status>>8))
		data = append(data, 0, 0)
	}

	return c.writePacket(data)
}

func (c *Conn) writePacket(data []byte) error {
	return c.pkg.WritePacket(data)
}

func (c *Conn) readPacket() ([]byte, error) {
	return c.pkg.ReadPacket()
}

func (c *Conn) close() error {
	err := c.rwc.Close()
	return err
}
