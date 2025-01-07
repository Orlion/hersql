package sidecar

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/Orlion/hersql/log"
	"github.com/Orlion/hersql/mysql"
	mysql_driver "github.com/go-sql-driver/mysql"
)

var DEFAULT_CAPABILITY uint32 = mysql.CLIENT_LONG_PASSWORD | mysql.CLIENT_LONG_FLAG |
	mysql.CLIENT_CONNECT_WITH_DB | mysql.CLIENT_PROTOCOL_41 |
	mysql.CLIENT_TRANSACTIONS | mysql.CLIENT_SECURE_CONNECTION | mysql.CLIENT_PLUGIN_AUTH

type Conn struct {
	connId          uint32
	server          *Server
	rwc             net.Conn
	remoteAddr      string
	pkg             *mysql.PacketIO
	salt            []byte
	status          uint16
	capability      uint32
	collation       uint8
	dsn             *mysql_driver.Config
	transportRunid  string
	transportConnId uint64
}

func (c *Conn) serve() {
	defer func() {
		if err := recover(); err != nil {
			log.Errorw("conn serve panic", "conn", c.name(), "error", err)
		}

		if c.transportConnId > 0 {
			if err := c.transportDisconnect(); err != nil {
				log.Warnw("conn serve transportDisconnect error occurred", "conn", c.name(), "error", err.Error())
			}
		}
		if err := c.close(); err != nil {
			log.Warnw("conn serve close error occurred ", "conn", c.name(), "error", err.Error())
		} else {
			log.Infow("conn serve closed", "conn", c.name())
		}
	}()

	log.Infow("conn serve start", "conn", c.name())

	err := c.handshake()
	if err != nil {
		log.Errorw("conn serve handshake error occurred", "conn", c.name(), "error", err.Error())
		return
	}

	log.Infow("conn serve handshake success", "conn", c.name())

	for {
		if c.server.shuttingDown() {
			break
		}

		data, err := c.readPacket()
		if err != nil {
			if errors.Is(err, io.EOF) {
				log.Infow("conn serve read closed", "conn", c.name())
			} else {
				log.Warnw("conn serve read packet error occurred", "conn", c.name(), "error", err.Error())
			}
			break
		}

		log.Infow("conn serve read packet", "conn", c.name(), "length", len(data))

		// 发送到服务端
		if responsePackets, err := c.transport(data); err != nil {
			log.Errorw("conn serve transport error occurred", "conn", c.name(), "error", err.Error())
			c.writeError(err)
		} else {
			if len(responsePackets) < 1 {
				c.transportConnId = 0
				log.Infow("conn serve transport closed", "conn", c.name())
				break
			}

			for _, responsePacket := range responsePackets {
				if err = c.writePacket(append(make([]byte, 4, 4+len(responsePacket)), responsePacket...)); err != nil {
					log.Errorw("conn serve write packet error occurred", "conn", c.name(), "error", err.Error())
					break
				}
				log.Infow("conn serve write packet", "conn", c.name(), "length", len(responsePacket))
			}
		}

		c.pkg.Sequence = 0
	}
}

func (c *Conn) handshake() error {
	if err := c.writeInitialHandshake(); err != nil {
		return fmt.Errorf("writeInitialHandshake error: %w", err)
	}

	if err := c.readHandshakeResponse(); err != nil {
		c.writeError(err)
		return fmt.Errorf("readHandshakeResponse error: %w", err)
	}

	if err := c.transportConnect(); err != nil {
		err = fmt.Errorf("transportConnect error: %w", err)
		c.writeError(err)
		return err
	}

	if err := c.writeOK(nil); err != nil {
		return fmt.Errorf("writeOK error: %w", err)
	}

	c.pkg.Sequence = 0

	return nil
}

func (c *Conn) writeInitialHandshake() error {
	data := make([]byte, 4, 128)

	//protocol version Always 10
	data = append(data, 10)

	//exit version[00]
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
	data = append(data, []byte(mysql.MysqlNativePassword)...)
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

	c.collation = data[pos]
	pos++

	//skip reserved 23[00]
	pos += 23

	//user name
	user := string(data[pos : pos+bytes.IndexByte(data[pos:], 0)])
	pos += len(user) + 1

	//auth length and auth
	authLen := int(data[pos])
	pos++

	pos += authLen

	if c.capability&mysql.CLIENT_CONNECT_WITH_DB > 0 {
		if len(data[pos:]) == 0 {
			return errors.New(`the database must be specified as a dsn in the format "[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]"`)
		}

		dsnStr := string(data[pos : pos+bytes.IndexByte(data[pos:], 0)])

		c.dsn, err = mysql_driver.ParseDSN(dsnStr)
		if err != nil {
			return fmt.Errorf(`the database "%s" failed to be parsed as a dsn, error: %w. the correct format is "[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]"`, dsnStr, err)
		}

		pos += len(dsnStr) + 1
	} else {
		return errors.New(`the database must be specified as a dsn in the format "[username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]"`)
	}

	return nil
}

func (c *Conn) writeOK(r *mysql.Result) error {
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

func (c *Conn) writeError(e error) error {
	var m *mysql.SqlError
	var ok bool
	if m, ok = e.(*mysql.SqlError); !ok {
		m = mysql.NewError(mysql.ER_UNKNOWN_ERROR, e.Error())
	}

	data := make([]byte, 4, 16+len(m.Message))

	data = append(data, mysql.ERR_HEADER)
	data = append(data, byte(m.Code), byte(m.Code>>8))

	if c.capability&mysql.CLIENT_PROTOCOL_41 > 0 {
		data = append(data, '#')
		data = append(data, m.State...)
	}

	data = append(data, m.Message...)

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

func (c *Conn) name() string {
	return fmt.Sprintf("[id:%d, from:%s, transportConnid:%d]", c.connId, c.remoteAddr, c.transportConnId)
}
