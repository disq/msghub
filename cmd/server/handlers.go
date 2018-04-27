package main

import (
	"net"
	"sync/atomic"
)

func (s *Server) handleConn(conn net.Conn) {
	s.wg.Add(1)
	defer s.wg.Done()

	id := atomic.AddUint64(&s.lastSessionId, 1)

	s.logger.Printf("[%v] connected (%v)", id, conn.RemoteAddr().String())

	c := Session{
		ID:   id,
		conn: conn,
	}

	s.sessMu.Lock()
	s.sess = append(s.sess, &c)
	s.sessMu.Unlock()

	// TODO
}
