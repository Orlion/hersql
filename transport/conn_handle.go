package transport

import (
	"fmt"

	"github.com/Orlion/hersql/log"
	"github.com/Orlion/hersql/mysql"
)

func (c *Conn) transport(packet []byte) ([][]byte, error) {
	c.pkg.Sequence = 0
	if err := c.writePacket(append(make([]byte, 4, 4+len(packet)), packet...)); err != nil {
		return nil, err
	}

	cmd := packet[0]
	switch cmd {
	case mysql.COM_PING:
	case mysql.COM_INIT_DB:
	case mysql.COM_QUERY:
		return c.handleQuery()
	case mysql.COM_QUIT:
		return c.handleQuit()
	case mysql.COM_FIELD_LIST:
		return c.handleFieldList()
	default:
		return nil, mysql.NewError(mysql.ER_UNKNOWN_ERROR, fmt.Sprintf("command %d not supported now", cmd))
	}

	return nil, nil
}

func (c *Conn) handleQuery() ([][]byte, error) {
	// https://dev.mysql.com/doc/dev/mysql-server/latest/page_protocol_com_query_response.html
	packets := make([][]byte, 0)
	hasEof := false
Loop:
	for {
		packet, err := c.readPacket()
		if err != nil {
			return nil, err
		}

		packets = append(packets, packet)

		switch packet[0] {
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

	return packets, nil
}

func (c *Conn) handleFieldList() ([][]byte, error) {
	// https://dev.mysql.com/doc/dev/mysql-server/latest/page_protocol_com_field_list.html
	packets := make([][]byte, 0)
Loop:
	for {
		packet, err := c.readPacket()
		if err != nil {
			return nil, err
		}

		packets = append(packets, packet)

		switch packet[0] {
		case mysql.OK_HEADER:
			c.server.delConn(c.id)
			c.close()
			log.Panicw("conn handle field list received an OK packet while parsing the COM_FIELD_LIST response")
		case mysql.EOF_HEADER:
			// eof
			break Loop
		case mysql.ERR_HEADER:
			// error
			break Loop
		default:
			// Column Definition
		}
	}

	return packets, nil
}

func (c *Conn) handleQuit() ([][]byte, error) {
	c.server.delConn(c.id)
	c.close()
	return nil, nil
}
