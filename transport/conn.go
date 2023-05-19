package transport

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Orlion/hersql/mysql"
)

type Conn struct {
	id         uint64
	rwc        net.Conn
	createAt   time.Time
	pkg        *mysql.PacketIO
	salt       []byte
	capability uint32
	status     uint16
	user       string
	passwd     string
	dbname     string
	collation  mysql.CollationId
}

func (c *Conn) handshake() error {
	if err := c.readInitialHandshake(); err != nil {
		return err
	}

	if err := c.writeHandshakeResponse(); err != nil {
		return err
	}

	if _, err := c.readOK(); err != nil {
		return err
	}

	c.pkg.Sequence = 0

	return nil
}

func (c *Conn) readInitialHandshake() error {
	data, err := c.readPacket()
	if err != nil {
		return err
	}

	if data[0] == mysql.ERR_HEADER {
		return errors.New("read initial handshake error")
	}

	if data[0] < mysql.MinProtocolVersion {
		return fmt.Errorf("invalid protocol version %d, must >= 10", data[0])
	}

	//skip mysql version and connection id
	//mysql version end with 0x00
	//connection id length is 4
	pos := 1 + bytes.IndexByte(data[1:], 0x00) + 1 + 4

	c.salt = append(c.salt, data[pos:pos+8]...)

	//skip filter
	pos += 8 + 1

	//capability lower 2 bytes
	c.capability = uint32(binary.LittleEndian.Uint16(data[pos : pos+2]))

	pos += 2

	if len(data) > pos {
		//skip exit charset
		//c.charset = data[pos]
		pos += 1

		c.status = binary.LittleEndian.Uint16(data[pos : pos+2])
		pos += 2

		c.capability = uint32(binary.LittleEndian.Uint16(data[pos:pos+2]))<<16 | c.capability

		pos += 2

		//skip auth data len or [00]
		//skip reserved (all [00])
		pos += 10 + 1

		// The documentation is ambiguous about the length.
		// The official Python library uses the fixed length 12
		// mysql-entrance also use 12
		// which is not documented but seems to work.
		c.salt = append(c.salt, data[pos:pos+12]...)
	}

	return nil
}

func (c *Conn) writeHandshakeResponse() error {
	// Adjust exit capability flags based on exit support
	capability := mysql.CLIENT_PROTOCOL_41 | mysql.CLIENT_SECURE_CONNECTION |
		mysql.CLIENT_LONG_PASSWORD | mysql.CLIENT_TRANSACTIONS | mysql.CLIENT_LONG_FLAG

	capability &= c.capability

	//packet length
	//capbility 4
	//max-packet size 4
	//charset 1
	//reserved all[0] 23
	length := 4 + 4 + 1 + 23

	//username
	length += len(c.user) + 1

	//we only support secure connection
	auth := mysql.CalcPassword(c.salt, []byte(c.passwd))

	length += 1 + len(auth)

	if len(c.dbname) > 0 {
		capability |= mysql.CLIENT_CONNECT_WITH_DB

		length += len(c.dbname) + 1
	}

	c.capability = capability

	data := make([]byte, length+4)

	//capability [32 bit]
	data[4] = byte(capability)
	data[5] = byte(capability >> 8)
	data[6] = byte(capability >> 16)
	data[7] = byte(capability >> 24)

	//MaxPacketSize [32 bit] (none)
	//data[8] = 0x00
	//data[9] = 0x00
	//data[10] = 0x00
	//data[11] = 0x00

	//Charset [1 byte]
	data[12] = byte(c.collation)

	//Filler [23 bytes] (all 0x00)
	pos := 13 + 23

	//User [null terminated string]
	if len(c.user) > 0 {
		pos += copy(data[pos:], c.user)
	}
	//data[pos] = 0x00
	pos++

	// auth [length encoded integer]
	data[pos] = byte(len(auth))
	pos += 1 + copy(data[pos+1:], auth)

	// db [null terminated string]
	if len(c.dbname) > 0 {
		pos += copy(data[pos:], c.dbname)
		//data[pos] = 0x00
	}

	return c.writePacket(data)
}

func (c *Conn) readOK() (*mysql.Result, error) {
	data, err := c.readPacket()
	if err != nil {
		return nil, err
	}

	if data[0] == mysql.OK_HEADER {
		return c.handleOKPacket(data)
	} else if data[0] == mysql.ERR_HEADER {
		return nil, c.handleErrorPacket(data)
	} else {
		return nil, errors.New("invalid ok packet")
	}
}

func (c *Conn) handleOKPacket(data []byte) (*mysql.Result, error) {
	var n int
	var pos int = 1

	r := new(mysql.Result)

	r.AffectedRows, _, n = mysql.LengthEncodedInt(data[pos:])
	pos += n
	r.InsertId, _, n = mysql.LengthEncodedInt(data[pos:])
	pos += n

	if c.capability&mysql.CLIENT_PROTOCOL_41 > 0 {
		r.Status = binary.LittleEndian.Uint16(data[pos:])
		c.status = r.Status
		pos += 2
	} else if c.capability&mysql.CLIENT_TRANSACTIONS > 0 {
		r.Status = binary.LittleEndian.Uint16(data[pos:])
		c.status = r.Status
		pos += 2
	}

	return r, nil
}

func (c *Conn) handleErrorPacket(data []byte) error {
	e := new(mysql.SqlError)

	var pos int = 1

	e.Code = binary.LittleEndian.Uint16(data[pos:])
	pos += 2

	if c.capability&mysql.CLIENT_PROTOCOL_41 > 0 {
		//skip '#'
		pos++
		e.State = string(data[pos : pos+5])
		pos += 5
	}

	e.Message = string(data[pos:])

	return e
}

func (c *Conn) readPacket() ([]byte, error) {
	return c.pkg.ReadPacket()
}

func (c *Conn) writePacket(data []byte) error {
	return c.pkg.WritePacket(data)
}

func (c *Conn) close() error {
	return c.rwc.Close()
}

func (c *Conn) transport(packet []byte) ([][]byte, error) {
	c.pkg.Sequence = 0
	if err := c.writePacket(append(make([]byte, 4, 4+len(packet)), packet...)); err != nil {
		return nil, err
	}

	cmd := packet[0]
	switch cmd {
	case mysql.COM_QUIT:
		panic("")
	case mysql.COM_QUERY:
		return c.handleQuery(packet)
	case mysql.COM_PING:
	case mysql.COM_INIT_DB:
	case mysql.COM_FIELD_LIST:
	default:
		return nil, mysql.NewError(mysql.ER_UNKNOWN_ERROR, fmt.Sprintf("command %d not supported now, packet: %s", cmd, string(packet)))
	}

	return nil, nil
}

func (c *Conn) handleQuery(packet []byte) ([][]byte, error) {
	// https://dev.mysql.com/doc/dev/mysql-server/latest/page_protocol_com_query_response.html
	responsePackets := make([][]byte, 0)
	hasEof := false
Loop:
	for {
		responsePacket, err := c.readPacket()
		if err != nil {
			return nil, err
		}

		responsePackets = append(responsePackets, responsePacket)

		switch responsePacket[0] {
		case mysql.OK_HEADER:
			// ok
			break Loop
		case mysql.EOF_HEADER:
			// eof
			if hasEof {
				break Loop
			}
			hasEof = true
		case mysql.ERR_HEADER:
			// error
			break Loop
		default:
			// column_count | Field metadata | The row data
		}
	}

	return responsePackets, nil
}
