package main

import (
	"context"
	"net"
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

	sess   []*Session
	sessMu sync.RWMutex

	lastSessionId uint64
}

// Session stores each sessions own data
type Session struct {
	ID   uint64
	conn net.Conn
}

// NewServer creates a new Server instance
func NewServer(ctx context.Context, logger msghub.Logger) *Server {

	// Create a new context so that we can shut down goroutines without requiring the parent context to be cancelled
	ctx, cancel := context.WithCancel(ctx)
	return &Server{
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
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

	// Accept connections
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return err
		}

		// Never block
		go s.handleConn(conn)
	}

	return nil
}

func (s *Server) Close() {
	if s.listener != nil {
		s.listener.Close() // error ignored
	}

	s.cancel()
	s.wg.Wait()
}
