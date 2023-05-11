package agents

import (
	"net"
	"net/http"
	"sync/atomic"
)

var connId uint64

func genConnId() uint64 {
	return atomic.AddUint64(&connId, 1)
}

type Handler struct {
	conns map[int64]Conn
}

func (h *Handler) HandleConn(w http.ResponseWriter, r *http.Request) {
	host := r.PostFormValue("host")
	user := r.PostFormValue("user")
	password := r.PostFormValue("password")

	if err := h.newConn(host, user, password); err != nil {
		return
	}
}

func (h *Handler) HandleRequest(w http.ResponseWriter, r *http.Request) {
	packet := r.PostFormValue("packet")
	if err := h.conns[0].writePacket([]byte(packet)); err != nil {
		return
	}
}

func (h *Handler) newConn(host, user, password string) error {
	rwc, err := net.Dial("tcp", host)
	if err != nil {
		return err
	}

	conn := &Conn{
		id:  genConnId(),
		rwc: rwc,
	}

	if err := conn.handshake(); err != nil {
		return err
	}

	return nil
}
