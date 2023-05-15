package exit

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) HandleConnect(w http.ResponseWriter, r *http.Request) {
	addr := r.PostFormValue("addr")
	dbname := r.PostFormValue("dbname")
	user := r.PostFormValue("user")
	passwd := r.PostFormValue("passwd")

	rwc, err := net.Dial("tcp", addr)
	if err != nil {
		responseFail(w, fmt.Sprintf("exit create conn failed: %s", err.Error()))
		return
	}

	conn := &Conn{
		rwc:      rwc,
		createAt: time.Now(),
		dbname:   dbname,
		user:     user,
		passwd:   passwd,
	}
	if err := conn.handshake(); err != nil {
		responseFail(w, fmt.Sprintf("exit create conn failed: %s", err.Error()))
		return
	}

	connId := s.addConn(conn)
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, connId)
	responseSuccess(w, b)
}

func (s *Server) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	runid := r.PostFormValue("runid")
	if runid != s.runid {
		responseFail(w, fmt.Sprintf("connId %s parse error: %s", connIdStr, err.Error()))
		return
	}
	connIdStr := r.PostFormValue("connId")
	connId, err := strconv.ParseUint(connIdStr, 10, 64)
	if err != nil {
		responseFail(w, fmt.Sprintf("connId %s parse error: %s", connIdStr, err.Error()))
		return
	}

	conn, exists := s.conns[connId]
	if !exists {
		responseFail(w, fmt.Sprintf("connId %d not found", connId))
		return
	}

	if err = conn.close(); err != nil {
		// todo: log
	}
	delete(s.conns, connId)
}

func (h *Server) HandleTransport(w http.ResponseWriter, r *http.Request) {
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
