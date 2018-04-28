package main

import (
	"bufio"
	"context"
	"net"
	"strings"
	"sync"

	"github.com/disq/msghub"
)

// Server is our main struct
type Server struct {
	ctx    context.Context
	cancel context.CancelFunc
	logger msghub.Logger

	listener net.Listener
	wg       sync.WaitGroup

	sess   map[uint64]*Session
	sessMu sync.RWMutex

	lastSessionId uint64
}

// Session stores each sessions own data
type Session struct {
	ID      uint64
	WriteCh chan *Message

	ctx    context.Context // Overkill?
	cancel context.CancelFunc

	conn net.Conn

	reader *bufio.Reader
	writer *bufio.Writer
}

// NewServer creates a new Server instance
func NewServer(ctx context.Context, logger msghub.Logger) *Server {

	// Create a new context so that we can shut down goroutines without requiring the parent context to be cancelled
	ctx, cancel := context.WithCancel(ctx)
	return &Server{
		ctx:    ctx,
		cancel: cancel,
		logger: logger,

		sess: make(map[uint64]*Session),
	}
}

// Listen listens for TCP connections on the given ip:port
func (s *Server) Listen(listenAddr string) error {
	// Bind port
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	s.listener = listener

	s.logger.Printf("Listening on %v", listenAddr)

	go func() {
		<-s.ctx.Done()
		s.close()
	}()

	// Accept connections
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}

		// Never block
		go s.handleConn(conn) // FIXME(kh): Worker pool?
	}

	return nil
}

func (s *Server) close() {
	s.listener.Close()

	// Disconnect all clients
	s.sessMu.RLock()
	for _, c := range s.sess {
		c.Send(SystemMessage{"Server shutting down..."})
	}
	s.sessMu.RUnlock()

	s.cancel()
	s.wg.Wait()
}

// isErrNetClosing checks if the error is ErrNetClosing (defined in stdlib internal/poll/fd.go)
func isErrNetClosing(err error) bool {
	// No other way to check
	return strings.Contains(err.Error(), "use of closed network connection")
}
