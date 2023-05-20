package transport

import (
	"context"
	"net/http"
	"strconv"
	"sync"
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
		runid:      strconv.FormatInt(time.Now().UnixNano(), 10),
		Addr:       conf.Addr,
		conns:      make(map[uint64]*Conn),
		nextConnId: 1,
	}

	serveMux := http.NewServeMux()
	serveMux.HandleFunc("/connect", s.HandleConnect)
	serveMux.HandleFunc("/disconnect", s.HandleDisconnect)
	serveMux.HandleFunc("/transport", s.HandleTransport)
	s.http = &http.Server{
		Addr:         conf.Addr,
		Handler:      serveMux,
		ReadTimeout:  time.Duration(conf.ReadTimeoutMillis) * time.Millisecond,
		WriteTimeout: time.Duration(conf.WriteTimeoutMillis) * time.Millisecond,
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

func (s *Server) addConn(conn *Conn) uint64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn.id = s.nextConnId
	s.conns[conn.id] = conn

	s.nextConnId++

	return conn.id
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
