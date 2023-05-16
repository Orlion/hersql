package exit

import (
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/Orlion/hersql/log"
)

func (s *Server) HandleConnect(w http.ResponseWriter, r *http.Request) {
	addr := r.PostFormValue("addr")
	dbname := r.PostFormValue("dbname")
	user := r.PostFormValue("user")
	passwd := r.PostFormValue("passwd")

	rwc, err := net.Dial("tcp", addr)
	if err != nil {
		responseFail(w, fmt.Sprintf("exit dial to %s failed: %s", addr, err.Error()))
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

	log.Infof("conn %d connect to %s, from %s", connId, addr, r.RemoteAddr)

	responseSuccess(w, &ResponseConnectData{
		Runid:  s.runid,
		ConnId: connId,
	})
}

func (s *Server) HandleDisconnect(w http.ResponseWriter, r *http.Request) {
	runid := r.PostFormValue("runid")
	if runid != s.runid {
		responseFail(w, "the runid does not match, the server may have been restarted, please try to reconnect")
		return
	}
	connIdStr := r.PostFormValue("connId")
	connId, err := strconv.ParseUint(connIdStr, 10, 64)
	if err != nil {
		responseFail(w, fmt.Sprintf("conn %s parse error: %s", connIdStr, err.Error()))
		return
	}

	conn, exists := s.getConn(connId)
	if !exists {
		responseFail(w, fmt.Sprintf("conn %d not found", connId))
		return
	}

	s.delConn(connId)

	log.Infof("conn %d disconnect, from %s", connId, r.RemoteAddr)
	if err = conn.close(); err != nil {
		log.Errorf("conn %d close error: %s", connId, err.Error())
	}

	responseSuccess(w, nil)
}

func (s *Server) HandleTransport(w http.ResponseWriter, r *http.Request) {
	runid := r.PostFormValue("runid")
	if runid != s.runid {
		responseFail(w, "the runid does not match, the server may have been restarted, please try to reconnect")
		return
	}
	connIdStr := r.PostFormValue("connId")
	connId, err := strconv.ParseUint(connIdStr, 10, 64)
	if err != nil {
		responseFail(w, fmt.Sprintf("conn %s parse error: %s", connIdStr, err.Error()))
		return
	}

	conn, exists := s.conns[connId]
	if !exists {
		responseFail(w, fmt.Sprintf("conn %d not found", connId))
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
