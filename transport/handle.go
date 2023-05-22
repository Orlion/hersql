package transport

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/Orlion/hersql/log"
	"github.com/Orlion/hersql/mysql"
)

func (s *Server) HandleConnect(w http.ResponseWriter, r *http.Request) {
	addr := r.PostFormValue("addr")
	dbname := r.PostFormValue("dbname")
	user := r.PostFormValue("user")
	passwd := r.PostFormValue("passwd")
	collationStr := r.PostFormValue("collation")

	collation, err := strconv.ParseUint(collationStr, 10, 8)
	if err != nil {
		responseFail(w, fmt.Sprintf("handle connect parse collation %s error: %s", collationStr, err.Error()))
		return
	}

	rwc, err := net.Dial("tcp", addr)
	if err != nil {
		responseFail(w, fmt.Sprintf("handle connect dial to %s failed: %s", addr, err.Error()))
		return
	}

	conn := &Conn{
		rwc:       rwc,
		createAt:  time.Now(),
		server:    s,
		pkg:       mysql.NewPacketIO(rwc),
		dbname:    dbname,
		user:      user,
		passwd:    passwd,
		collation: uint8(collation),
	}
	if err := conn.handshake(); err != nil {
		rwc.Close()
		responseFail(w, fmt.Sprintf("handle connect create conn failed: %s", err.Error()))
		return
	}

	connId := s.addConn(conn)

	log.Infow("handle connect", "connId", connId, "addr", addr, "remoteAddr", r.RemoteAddr)

	connectResponse(w, s.runid, connId)
}

func (s *Server) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	runid := r.PostFormValue("runid")
	if runid != s.runid {
		responseFail(w, "handle disconnect the runid does not match, the server may have been restarted, please try to reconnect")
		return
	}
	connIdStr := r.PostFormValue("connId")
	connId, err := strconv.ParseUint(connIdStr, 10, 64)
	if err != nil {
		responseFail(w, fmt.Sprintf("handle disconnect conn %s parse error: %s", connIdStr, err.Error()))
		return
	}

	conn, exists := s.getConn(connId)
	if !exists {
		responseFail(w, fmt.Sprintf("handle disconnect conn %d not found", connId))
		return
	}

	s.delConn(connId)

	if err = conn.close(); err != nil {
		log.Errorw("handle disconnect conn close error occrred", "connId", connId, "error", err.Error())
	}

	log.Infow("handle disconnect", "connId", connId, "remoteAddr", r.RemoteAddr)

	disconnectResponse(w)
}

func (s *Server) HandleTransport(w http.ResponseWriter, r *http.Request) {
	runid := r.PostFormValue("runid")
	if runid != s.runid {
		responseFail(w, "handle transport the runid does not match, the server may have been restarted, please try to reconnect")
		return
	}
	connIdStr := r.PostFormValue("connId")
	connId, err := strconv.ParseUint(connIdStr, 10, 64)
	if err != nil {
		responseFail(w, fmt.Sprintf("handle transport connId %s parse error: %s", connIdStr, err.Error()))
		return
	}
	packet := r.PostFormValue("packet")

	conn, exists := s.getConn(connId)
	if !exists {
		responseFail(w, fmt.Sprintf("handle transport conn %d not found", connId))
		return
	}

	responsePackets, err := conn.transport([]byte(packet))
	if err != nil {
		responseFail(w, fmt.Sprintf("handle transport conn %d transport: %s", connId, err.Error()))
		return
	}

	log.Infow("handle transport", "connId", connId, "cmd", packet[0], "length", len(packet), "responsePacketsNum", len(responsePackets))

	transportResponse(w, responsePackets)
}

func (s *Server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w.Write([]byte(fmt.Sprintf("conn num: %d", len(s.conns))))
	for connId, conn := range s.conns {
		w.Write([]byte(fmt.Sprintf("connId: %d, conn: %s \n", connId, conn.toString())))
	}
}
