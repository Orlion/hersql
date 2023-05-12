package entrance

import (
	"context"
	"errors"
	"github.com/Orlion/hersql/log"
	"github.com/Orlion/hersql/mysql"
	"github.com/Orlion/hersql/pkg/atomicx"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

var (
	ErrServerClosed      = errors.New("server closed")
	shutdownPollInterval = 500 * time.Millisecond
	connId               uint32
)

func genConnId() uint32 {
	return atomic.AddUint32(&connId, 1)
}

type Server struct {
	Addr         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	inShutdown   atomicx.Bool
	mu           sync.Mutex
	listener     net.Listener
	doneChan     chan struct{}
	connNum      int64
	httpClient   *http.Client
	httpHost     string
}

func NewServer(conf *Config) *Server {
	return &Server{
		Addr:         conf.Addr,
		ReadTimeout:  time.Duration(conf.ReadTimeoutMillis) * time.Millisecond,
		WriteTimeout: time.Duration(conf.WriteTimeoutMillis) * time.Millisecond,
		httpClient: &http.Client{
			Timeout: time.Duration(conf.HttpTimeoutMillis) * time.Millisecond,
		},
	}
}

func (s *Server) ListenAndServe() (err error) {
	s.listener, err = net.Listen("tcp", s.Addr)
	if err != nil {
		return
	}

	return s.serve()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Info("server shutdown...")

	s.inShutdown.SetTrue()

	s.mu.Lock()
	defer s.mu.Unlock()

	lnerr := s.listener.Close()
	s.closeDoneChanLocked()

	ticker := time.NewTicker(shutdownPollInterval)
	defer ticker.Stop()
	for {
		if s.getConnNum() == 0 {
			log.Info("server exit")
			return lnerr
		}
		select {
		case <-ctx.Done():
			log.Info("server exit")
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Server) closeDoneChanLocked() {
	ch := s.getDoneChanLocked()
	select {
	case <-ch:
	default:
		close(ch)
	}
}

func (s *Server) serve() error {
	log.Infof("server listen on %s...", s.Addr)

	var tempDelay time.Duration // how long to sleep on accept failure

	for {
		rw, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.getDoneChan():
				return ErrServerClosed
			default:
			}

			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}

				time.Sleep(tempDelay)
				continue
			}

			return err
		}

		c := s.newConn(rw)
		log.Debugf("server new Conn from %s", rw.RemoteAddr().String())
		go func() {
			c.serve()
		}()
	}
}

func (s *Server) shuttingDown() bool {
	return s.inShutdown.Get()
}

func (s *Server) newConn(rwc net.Conn) *Conn {
	c := &Conn{
		pkg:    mysql.NewPacketIO(rwc),
		connId: genConnId(),
		salt:   mysql.RandomBuf(20),
		server: s,
		rwc:    rwc,
		status: mysql.SERVER_STATUS_AUTOCOMMIT,
	}

	s.incrConnNum()

	return c
}

func (s *Server) getDoneChan() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getDoneChanLocked()
}

func (s *Server) getDoneChanLocked() chan struct{} {
	if s.doneChan == nil {
		s.doneChan = make(chan struct{})
	}
	return s.doneChan
}

func (s *Server) getConnNum() int64 {
	return atomic.LoadInt64(&s.connNum)
}

func (s *Server) incrConnNum() {
	atomic.AddInt64(&s.connNum, 1)
}

func (s *Server) decrConnNum() {
	atomic.AddInt64(&s.connNum, -1)
}
