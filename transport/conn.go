package transport

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/Orlion/hersql/log"
	"github.com/Orlion/hersql/mysql"
)

type Conn struct {
	id         uint64
	rwc        net.Conn
	createAt   time.Time
	server     *Server
	pkg        *mysql.PacketIO
	capability uint32
	collation  uint8
	status     uint16
	user       string
	passwd     string
	dbname     string
}

func (c *Conn) toString() string {
	return fmt.Sprintf("id: %d, createAt: %s, capability: %d, collation: %d, status: %d, user: %s, dbname: %s", c.id, c.createAt.Format("2006-01-02 15:04:05"), c.capability, c.collation, c.status, c.user, c.dbname)
}

func (c *Conn) handshake() error {
	authData, plugin, err := c.readInitialHandshake()
	if err != nil {
		return fmt.Errorf("readInitialHandshake error: %w", err)
	}

	authResp, err := c.auth(authData, plugin)
	if err != nil {
		return fmt.Errorf("auth error: %w", err)
	}

	if err := c.writeHandshakeResponse(authResp); err != nil {
		return fmt.Errorf("writeHandshakeResponse error: %w", err)
	}

	// Handle response to auth packet, switch methods if possible
	if err = c.handleAuthResult(authData, plugin); err != nil {
		return fmt.Errorf("handleAuthResult error: %w", err)
	}

	if _, err := c.readOK(); err != nil {
		return fmt.Errorf("readOK error: %w", err)
	}

	c.pkg.Sequence = 0

	return nil
}

func (c *Conn) readInitialHandshake() (authData []byte, plugin string, err error) {
	data, err := c.readPacket()
	if err != nil {
		return
	}

	if data[0] == mysql.ERR_HEADER {
		return nil, "", errors.New("read initial handshake error")
	}

	if data[0] < mysql.MinProtocolVersion {
		return nil, "", fmt.Errorf("invalid protocol version %d, must >= 10", data[0])
	}

	//skip mysql version and connection id
	//mysql version end with 0x00
	//connection id length is 4
	pos := 1 + bytes.IndexByte(data[1:], 0x00) + 1 + 4

	authData = data[pos : pos+8]

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
		authData = append(authData, data[pos:pos+12]...)
		pos += 13

		if end := bytes.IndexByte(data[pos:], 0x00); end != -1 {
			plugin = string(data[pos : pos+end])
		} else {
			plugin = string(data[pos:])
		}
	}

	return
}

func (c *Conn) auth(authData []byte, plugin string) (authResp []byte, err error) {
	switch plugin {
	case mysql.MysqlNativePassword:
		authResp = mysql.ScramblePassword(authData, []byte(c.passwd))
	case mysql.CachingSha2Password:
		authResp = mysql.ScrambleSHA256Password(authData, []byte(c.passwd))
	default:
		err = newErrUnsupportedAuthPlugin(plugin)
	}

	return
}

func (c *Conn) writeHandshakeResponse(authResp []byte) error {
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

	length += 1 + len(authResp)

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
	data[pos] = byte(len(authResp))
	pos += 1 + copy(data[pos+1:], authResp)

	// db [null terminated string]
	if len(c.dbname) > 0 {
		pos += copy(data[pos:], c.dbname)
		//data[pos] = 0x00
	}

	return c.writePacket(data)
}

func (c *Conn) handleAuthResult(oldAuthData []byte, plugin string) error {
	authData, newPlugin, err := c.readAuthResult()
	if err != nil {
		return fmt.Errorf("readAuthResult1 error: %w", err)
	}

	log.Debugw("handleAuthResult", "authData", authData, "newPlugin", newPlugin, "plugin", plugin)

	if newPlugin != "" {
		if authData == nil {
			authData = oldAuthData
		} else {
			copy(oldAuthData, authData)
		}

		plugin = newPlugin

		authResp, err := c.auth(authData, plugin)
		if err != nil {
			return fmt.Errorf("auth error: %w", err)
		}

		if err := c.writeAuthSwitchPacket(authResp); err != nil {
			return fmt.Errorf("writeAuthSwitchPacket error: %w", err)
		}

		authData, newPlugin, err = c.readAuthResult()
		if err != nil {
			return fmt.Errorf("readAuthResult2 error: %w", err)
		}

		if newPlugin != "" {
			return ErrMalformPkt
		}
	}

	switch plugin {
	case mysql.CachingSha2Password:
		switch len(authData) {
		case 0:
			return nil // auth successful
		case 1:
			switch authData[0] {
			case mysql.CachingSha2PasswordFastAuthSuccess:
				_, err = c.readOK()
				return err
			case mysql.CachingSha2PasswordPerformFullAuthentication:
				data := make([]byte, 5)
				data[4] = mysql.CachingSha2PasswordRequestPublicKey
				if err = c.writePacket(data); err != nil {
					return err
				}

				if data, err = c.readPacket(); err != nil {
					return err
				}

				if data[0] != mysql.AUTH_MORE_DATA_HEADER {
					return errors.New("unexpected resp from server for caching_sha2_password, perform full authentication")
				}

				// parse public key
				block, rest := pem.Decode(data[1:])
				if block == nil {
					return fmt.Errorf("no pem data found, data: %s", rest)
				}
				pkix, err := x509.ParsePKIXPublicKey(block.Bytes)
				if err != nil {
					return err
				}
				pubKey := pkix.(*rsa.PublicKey)

				// send encrypted password
				err = c.sendEncryptedPassword(oldAuthData, pubKey)
				if err != nil {
					return err
				}

				_, err = c.readOK()
				return err
			default:
				return ErrMalformPkt
			}
		default:
			return ErrMalformPkt
		}

	default:
		return nil // auth successful
	}
}

func (c *Conn) sendEncryptedPassword(seed []byte, pub *rsa.PublicKey) error {
	enc, err := mysql.EncryptPassword(c.passwd, seed, pub)
	if err != nil {
		return err
	}
	return c.writeAuthSwitchPacket(enc)
}

func (c *Conn) readAuthResult() ([]byte, string, error) {
	data, err := c.readPacket()
	if err != nil {
		return nil, "", fmt.Errorf("readPacket error: %w", err)
	}

	switch data[0] {

	case mysql.OK_HEADER:
		_, err = c.handleOKPacket(data)
		return nil, "", err

	case mysql.AUTH_MORE_DATA_HEADER:
		return data[1:], "", nil

	case mysql.EOF_HEADER:
		if len(data) == 1 {
			// https://dev.mysql.com/doc/internals/en/connection-phase-packets.html#packet-Protocol::OldAuthSwitchRequest
			return nil, mysql.MysqlOldPassword, nil
		}
		pluginEndIndex := bytes.IndexByte(data, 0x00)
		if pluginEndIndex < 0 {
			return nil, "", ErrMalformPkt
		}
		plugin := string(data[1:pluginEndIndex])
		authData := data[pluginEndIndex+1:]
		return authData, plugin, nil

	case mysql.ERR_HEADER:
		return nil, "", c.handleErrorPacket(data)

	default:
		return nil, "", ErrMalformPkt
	}
}

func (c *Conn) writeAuthSwitchPacket(authData []byte) error {
	data := make([]byte, 4+len(authData))
	// Add the auth data [EOF]
	copy(data[4:], authData)
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
