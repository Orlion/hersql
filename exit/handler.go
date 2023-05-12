package exit

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
)

var connId uint64

func genConnId() uint64 {
	return atomic.AddUint64(&connId, 1)
}

type Handler struct {
	conns map[uint64]*Conn
}

func NewHandler() *Handler {
	return &Handler{
		conns: make(map[uint64]*Conn),
	}
}

func (h *Handler) HandleConnect(w http.ResponseWriter, r *http.Request) {
	host := r.PostFormValue("host")
	db := r.PostFormValue("db")
	user := r.PostFormValue("user")
	password := r.PostFormValue("password")

	connId, err := h.newConn(host, db, user, password)
	if err != nil {
		responseFail(w, fmt.Sprintf("exit create conn failed: %s", err.Error()))
		return
	}

	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, connId)
	responseSuccess(w, b)
}

func (h *Handler) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	connIdStr := r.PostFormValue("connId")
	connId, err := strconv.ParseUint(connIdStr, 10, 64)
	if err != nil {
		responseFail(w, fmt.Sprintf("connId %s parse error: %s", connIdStr, err.Error()))
		return
	}

	conn, exists := h.conns[connId]
	if !exists {
		responseFail(w, fmt.Sprintf("connId %d not found", connId))
		return
	}

	if err = conn.close(); err != nil {
		// todo: log
	}
	delete(h.conns, connId)
}

func (h *Handler) HandleTransport(w http.ResponseWriter, r *http.Request) {
	connIdStr := r.PostFormValue("connId")
	connId, err := strconv.ParseUint(connIdStr, 10, 64)
	if err != nil {
		responseFail(w, fmt.Sprintf("connId %s parse error: %s", connIdStr, err.Error()))
		return
	}

	conn, exists := h.conns[connId]
	if !exists {
		responseFail(w, fmt.Sprintf("connId %d not found", connId))
		return
	}

	packet := r.PostFormValue("packet")
	if err := conn.writePacket([]byte(packet)); err != nil {
		responseFail(w, fmt.Sprintf("write packet error: %s", err.Error()))
		return
	}

	responsePacket, err := conn.readPacket()
	if err != nil {
		responseFail(w, fmt.Sprintf("read packet error: %s", err.Error()))
		return
	}

	responseSuccess(w, responsePacket)
}

func (h *Handler) newConn(host, db, user, password string) (uint64, error) {
	rwc, err := net.Dial("tcp", host)
	if err != nil {
		return 0, err
	}

	connId := genConnId()
	conn := &Conn{
		id:       connId,
		rwc:      rwc,
		db:       db,
		user:     user,
		password: password,
	}

	if err := conn.handshake(); err != nil {
		return 0, err
	}

	return connId, nil
}
