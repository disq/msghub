package server

import (
	"bufio"
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
	wg       sync.WaitGroup // wg for client goroutines

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

// New creates a new Server instance
func New(ctx context.Context, logger msghub.Logger) *Server {

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

	s.logger.Printf("Listening on %v", listener.Addr())

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
		go s.HandleConn(conn) // FIXME(kh): Worker pool?
	}
}

func (s *Server) close() {
	// Disconnect all clients by sending them a shutdown message
	s.sessMu.RLock()
	for _, c := range s.sess {
		c.Send(ShutdownMessage{&SystemMessage{"Server shutting down..."}})
	}
	s.sessMu.RUnlock()

	// Wait for all client goroutines to shut down
	s.wg.Wait()

	s.listener.Close()
}
