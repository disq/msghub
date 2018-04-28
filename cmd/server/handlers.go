package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
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

		conn:   conn,
		ctx:    ctx,
		cancel: cancel,

		reader: bufio.NewReader(conn),
		writer: bufio.NewWriter(conn),
	}

	s.sessMu.Lock()
	s.sess[id] = c
	s.sessMu.Unlock()

	// Listen for writes, write them to client
	go s.ListenWriter(c)

	// Show help text
	s.SendHelp(c)

	// Read client commands
	go s.ListenReader(c)

	<-c.ctx.Done()

	// Disconnected, cleanup
	s.logger.Printf("[%v] disconnected", id)

	s.sessMu.Lock()
	delete(s.sess, id)
	s.sessMu.Unlock()

	close(c.WriteCh)
	c.conn.Close()
}

// ListenWriter listens on the client's write channel and forwards messages to client
func (s *Server) ListenWriter(c *Session) {
	for {
		select {
		case <-c.ctx.Done():
			return
		case w := <-c.WriteCh:
			c.writer.WriteString(w + "\n")
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
			if err != nil && (err == io.EOF || isErrNetClosing(err)) {
				return
			} else if err != nil {
				s.logger.Printf("[%v] Error reading: %v", c.ID, err)
				continue
			}

			if err := s.ParseHandleCommand(c, ln); err != nil {
				s.logger.Printf("[%v] Error handling: %v", c.ID, err)
				c.WriteCh <- fmt.Sprintf("Error in command: %v", err)
			}

		}

	}
}

func (s *Server) ParseHandleCommand(c *Session, ln string) error {
	ln = strings.TrimSpace(ln)
	lnParts := strings.SplitN(ln, " ", 3)

	switch lnParts[0] {

	// Allow newlines
	case "":
		return nil

	// Ask for id
	case "a":
		c.WriteCh <- fmt.Sprintf("%v", c.ID)
		return nil

	// Ask for clients
	case "w":
		list := s.GetConnectedSessions()
		var flist []string
		for _, sessionId := range list {
			if sessionId == c.ID {
				continue
			}
			flist = append(flist, strconv.FormatUint(sessionId, 10))
		}
		c.WriteCh <- fmt.Sprintf("%v", strings.Join(flist, " "))
		return nil

	// Broadcast message
	case "s":
		if len(lnParts) != 3 {
			return fmt.Errorf("Invalid parameters")
		}

		// Parse
		dstList, err := s.ValidateBroadcastDestinations(c, lnParts[1])
		if err != nil {
			return err
		}
		if len(dstList) == 0 {
			return fmt.Errorf("No clients specified")
		}
		for d := range dstList {
			err = s.SendMessage(d, lnParts[2])
			if err != nil {
				return err
			}
		}
		c.WriteCh <- fmt.Sprintf("Sent to %v clients", len(dstList))
		return nil

	// Disconnect
	case "d":
		c.WriteCh <- "Disconnecting..."
		c.cancel()
		return nil

	case "?":
		s.SendHelp(c)
		return nil
	}

	return fmt.Errorf("Unhandled command")
}

func (s *Server) GetConnectedSessions() []uint64 {
	var ret []uint64

	s.sessMu.RLock()
	for k := range s.sess {
		ret = append(ret, k)
	}
	s.sessMu.RUnlock()

	return ret
}

func (s *Server) SendHelp(c *Session) {
	c.WriteCh <- `
Welcome! Commands:
w  ask for clients
s  broadcast message (example: s 1,2 message)
a  ask for id
d  disconnect
?  this message
----
`
}
