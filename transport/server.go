package transport

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Orlion/hersql/log"
)

type Server struct {
	mu         sync.Mutex
	runid      string
	Addr       string
	http       *http.Server
	nextConnId uint64
	conns      map[uint64]*Conn
}

func NewServer(conf *Config) *Server {
	withDefaultConf(conf)

	s := &Server{
		runid: strconv.FormatInt(time.Now().UnixNano(), 10),
		Addr:  conf.Addr,
		conns: make(map[uint64]*Conn),
	}

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/connect", s.HandleConnect)
	serveMux.HandleFunc("/disconnect", s.HandleDisconnect)
	serveMux.HandleFunc("/transport", s.HandleTransport)
	serveMux.HandleFunc("/status", s.HandleStatus)
	s.http = &http.Server{
		Addr:    conf.Addr,
		Handler: serveMux,
	}

	return s
}

func (s *Server) ListenAndServe() error {
	log.Infow("server serve", "addr", s.Addr)
	return s.http.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Infow("server shutdown...")
	err := s.http.Shutdown(ctx)

	s.mu.Lock()
	defer s.mu.Unlock()

	for connId, conn := range s.conns {
		delete(s.conns, connId)
		conn.close()
	}

	return err
}

func (s *Server) addConn(conn *Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.conns[conn.id] = conn
}

func (s *Server) genConnId() uint64 {
	return atomic.AddUint64(&s.nextConnId, 1)
}

func (s *Server) delConn(connId uint64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.conns, connId)
}

func (s *Server) getConn(connId uint64) (*Conn, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	conn, exists := s.conns[connId]
	return conn, exists
}
