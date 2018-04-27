package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync/atomic"
)

const chanBufferSize = 10

// handleConn manages a connection's lifecycle
func (s *Server) handleConn(conn net.Conn) {
	s.wg.Add(1)
	defer s.wg.Done()

	id := atomic.AddUint64(&s.lastSessionId, 1)

	s.logger.Printf("[%v] connected (%v)", id, conn.RemoteAddr().String())

	ctx, cancel := context.WithCancel(s.ctx)

	c := &Session{
		ID:      id,
		WriteCh: make(chan string, chanBufferSize),

		conn: conn,
		ctx:  ctx,

		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}

	s.sessMu.Lock()
	s.sess[id] = c
	s.sessMu.Unlock()

	// Listen for writes, write them to client
	go s.ListenWriter(c)

	c.WriteCh <- `
Welcome! Commands:
w  ask for clients
s  broadcast message (example: s 1,2 message)
a  ask for id
d  disconnect
----
`

	// Read client commands
	s.ListenReader(c)
	cancel()

	// Disconnected, cleanup
	s.logger.Printf("[%v] disconnected", id)

	s.sessMu.Lock()
	delete(s.sess, id)
	s.sessMu.Unlock()
}

// ListenWriter listens on the client's write channel and forwards messages to client
func (s *Server) ListenWriter(c *Session) {
	for {
		select {
		case <-c.ctx.Done():
			return
		case w := <-c.WriteCh:
			c.writer.WriteString(w)
			c.writer.Flush()
		}
	}
}

func (s *Server) ListenReader(c *Session) {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			ln, err := c.reader.ReadString('\n')
			if err != nil && err == io.EOF {
				return
			} else if err != nil {
				s.logger.Printf("[%v] Error reading: %v", c.ID, err)
				continue
			}

			if err := s.ParseHandleCommand(c, ln); err != nil {
				s.logger.Printf("[%v] Error handling: %v", c.ID, err)
				c.WriteCh <- "Error in command\n"
			}

		}

	}
}

func (s *Server) ParseHandleCommand(c *Session, ln string) error {
	ln = strings.TrimSpace(ln)
	lnParts := strings.SplitN(ln, " ", 3)

	switch lnParts[0] {
	case "a":
		c.WriteCh <- fmt.Sprintf("%v\n", c.ID)
		return nil

		// TODO(kh) handle more commands
	}

	return fmt.Errorf("Unhandled command")
}
