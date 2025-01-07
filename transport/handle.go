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
		responseFail(w, fmt.Sprintf("handleConnect parse collation %s error: %s", collationStr, err.Error()))
		return
	}

	rwc, err := net.Dial("tcp", addr)
	if err != nil {
		responseFail(w, fmt.Sprintf("handleConnect dial to %s failed: %s", addr, err.Error()))
		return
	}

	conn := &Conn{
		id:        s.genConnId(),
		rwc:       rwc,
		createAt:  time.Now(),
		server:    s,
		pkg:       mysql.NewPacketIO(rwc),
		dbname:    dbname,
		user:      user,
		passwd:    passwd,
		collation: uint8(collation),
	}

	log.Infow("handleConnect create conn", "connId", conn.id, "addr", addr, "dbname", dbname, "user", user, "collation", collation)

	if err := conn.handshake(); err != nil {
		rwc.Close()
		responseFail(w, fmt.Sprintf("handleConnect handshake failed: %s", err.Error()))
		return
	}

	s.addConn(conn)

	log.Infow("handleConnect success", "connId", conn.id, "addr", addr, "remoteAddr", r.RemoteAddr)

	connectResponse(w, s.runid, conn.id)
}

func (s *Server) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	runid := r.PostFormValue("runid")
	if runid != s.runid {
		responseFail(w, "handleDisconnect the runid does not match, the server may have been restarted, please try to reconnect")
		return
	}
	connIdStr := r.PostFormValue("connId")
	connId, err := strconv.ParseUint(connIdStr, 10, 64)
	if err != nil {
		responseFail(w, fmt.Sprintf("handleDisconnect conn %s parse error: %s", connIdStr, err.Error()))
		return
	}

	conn, exists := s.getConn(connId)
	if !exists {
		responseFail(w, fmt.Sprintf("handleDisconnect conn %d not found", connId))
		return
	}

	s.delConn(connId)

	if err = conn.close(); err != nil {
		log.Errorw("handleDisconnect conn close error occrred", "connId", connId, "error", err.Error())
	}

	log.Infow("handleDisconnect success", "connId", connId, "remoteAddr", r.RemoteAddr)

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
		responseFail(w, fmt.Sprintf("handle transport conn %d not found, please try reconnect", connId))
		return
	}

	responsePackets, err := conn.transport([]byte(packet))
	if err != nil {
		responseFail(w, fmt.Sprintf("handle transport conn %d transport: %s", connId, err.Error()))
		return
	}

	log.Infow("handle transport", "connId", connId, "cmd", mysql.Cmd2Str(packet[0]), "length", len(packet), "responsePacketsNum", len(responsePackets))

	transportResponse(w, responsePackets)
}

func (s *Server) HandleStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	w.Write([]byte(fmt.Sprintf("conn num: %d", len(s.conns))))
	for connId, conn := range s.conns {
		w.Write([]byte(fmt.Sprintf("connId: %d, conn: %s \n", connId, conn.name())))
	}
}
