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
	"time"
)

const (
	chanBufferSize = 10
	readTimeout    = 5 * time.Second
)

// handleConn manages a connection's lifecycle
func (s *Server) handleConn(conn net.Conn) {
	s.wg.Add(1)
	defer s.wg.Done()

	id := atomic.AddUint64(&s.lastSessionId, 1)

	s.logger.Printf("[%v] connected (%v)", id, conn.RemoteAddr().String())

	// Separate context because we want to send shutdown message to clients on shutdown
	ctx, cancel := context.WithCancel(context.Background())

	c := &Session{
		ID:      id,
		WriteCh: make(chan *Message, chanBufferSize),

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
	if s.ctx.Err() != nil {
		s.logger.Printf("[%v] disconnected (shutting down)", id)
	} else {
		s.logger.Printf("[%v] disconnected", id)
	}

	s.sessMu.Lock()
	delete(s.sess, id)
	s.sessMu.Unlock()

	close(c.WriteCh)
	c.conn.Close()
}

// ListenWriter listens on the client's write channel and forwards messages to client
func (s *Server) ListenWriter(c *Session) {
	for w := range c.WriteCh {
		c.writer.WriteString((*w).Read() + "\n")
		c.writer.Flush()
	}
}

// ListenReader listens the bufio.reader of client session c, and evaluates input
func (s *Server) ListenReader(c *Session) {
	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			// Set timeout before reading to detect disconnections
			c.conn.SetReadDeadline(time.Now().Add(readTimeout))

			ln, err := c.reader.ReadString('\n')
			if err != nil && (err == io.EOF || isErrNetClosing(err)) {
				c.cancel()
				return
			} else if err != nil && isTimeoutError(err) {
				// deadline reached
				continue
			} else if err != nil {
				s.logger.Printf("[%v] Error reading: %v", c.ID, err)
				continue
			}

			if err := s.ParseHandleCommand(c, ln); err != nil {
				s.logger.Printf("[%v] Error handling: %v", c.ID, err)
				c.Send(CommandMessage{fmt.Sprintf("Error in command: %v", err)})
			}

		}

	}
}

// ParseHandleCommand evaluates a single command input line originating from session and handles the command
func (s *Server) ParseHandleCommand(c *Session, ln string) error {
	ln = strings.TrimSpace(ln)
	lnParts := strings.SplitN(ln, " ", 3)

	switch lnParts[0] {

	// Allow newlines
	case "":
		return nil

	// Ask for id
	case "a":
		c.Send(CommandMessage{fmt.Sprintf("%v", c.ID)})
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
		if len(flist) == 0 {
			c.Send(CommandMessage{"No other clients seem to have connected."})
		} else {
			c.Send(CommandMessage{fmt.Sprintf("List of clients: %v", strings.Join(flist, ","))})
		}
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

		msg := ClientMessage{c.ID, lnParts[2]}
		for d := range dstList {
			err = s.SendToSession(d, msg)
			if err != nil {
				return err
			}
		}

		if len(dstList) == 1 {
			c.Send(CommandMessage{"Sent to 1 client"})
		} else {
			c.Send(CommandMessage{fmt.Sprintf("Sent to %v clients", len(dstList))})
		}
		return nil

	// Disconnect
	case "d":
		c.Send(CommandMessage{"Disconnecting..."})
		c.cancel()
		return nil

	case "?":
		s.SendHelp(c)
		return nil
	}

	return fmt.Errorf("Unhandled command")
}

// GetConnectedSessions returns session ids of currently connected clients
func (s *Server) GetConnectedSessions() []uint64 {
	var ret []uint64

	s.sessMu.RLock()
	for k := range s.sess {
		ret = append(ret, k)
	}
	s.sessMu.RUnlock()

	return ret
}

// SendHelp shows a help message (which doubles as a welcome message) to client
func (s *Server) SendHelp(c *Session) {
	c.Send(CommandMessage{`
Welcome! Commands:
w  ask for clients
s  broadcast message (example: s 1,2 message)
a  ask for id
d  disconnect
?  this message
----
`})
}

// Send sends a Message to client
func (c *Session) Send(msg Message) error {
	select {
	case <-c.ctx.Done():
		return fmt.Errorf("Client %v not connected", c.ID)
	case c.WriteCh <- &msg:
	}

	return nil
}

// SendToSession sends message to the session with the given id
func (s *Server) SendToSession(id uint64, msg Message) error {
	s.sessMu.RLock()
	dst, ok := s.sess[id]
	s.sessMu.RUnlock()

	if !ok {
		return fmt.Errorf("Client %v not connected", id)
	}

	return dst.Send(msg)
}
