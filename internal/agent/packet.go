package agent

import (
	"errors"
	"io"
)

const PacketMaxSize = 0xffffff

type Packet struct {
	PayloadLength uint32
	SequenceId    byte
	Payload       []byte
}

func (c *conn) readPacket() (p *Packet, err error) {
	header := make([]byte, 4)
	n, err := io.ReadFull(c.rwc, header)
	if err != nil {
		return
	}

	if n != 4 {
		err = errors.New("header length is less than 4 bytes")
		return
	}

	payloadLength := uint32(header[0]) | uint32(header[1])<<8 | uint32(header[2])<<16
	payload := make([]byte, payloadLength)

	if payloadLength >= PacketMaxSize {
		total := 0
		for {
			n, err = io.ReadFull(c.rwc, payload[:PacketMaxSize])
			if err != nil {
				return
			}
			total += n
			if uint32(total) == payloadLength {
				break
			}
		}
	} else {
		_, err = io.ReadFull(c.rwc, payload)
		if err != nil {
			return
		}
	}

	p = &Packet{
		PayloadLength: payloadLength,
		SequenceId:    header[3],
		Payload:       payload,
	}

	return
}
