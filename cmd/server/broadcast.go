package main

import (
	"fmt"
	"strconv"
	"strings"
)

func (s *Server) ValidateBroadcastDestinations(c *Session, input string) (map[uint64]struct{}, error) {
	inParts := strings.Split(input, ",")

	slist := s.GetConnectedSessions()
	smap := make(map[uint64]struct{}, len(slist))
	for _, id := range slist {
		smap[id] = struct{}{}
	}

	dstList := make(map[uint64]struct{}, len(inParts))

	for _, dst := range inParts {
		dst = strings.TrimSpace(dst)
		dstId, err := strconv.ParseUint(dst, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Invalid destination %v: %v", dst, err)
		}
		if dstId == c.ID {
			return nil, fmt.Errorf("Can't send messages to yourself!")
		}
		if _, ok := smap[dstId]; !ok {
			return nil, fmt.Errorf("Invalid destination %v: Not connected", dst)
		}

		dstList[dstId] = struct{}{}
	}

	return dstList, nil
}

func (s *Server) SendMessage(id uint64, msg string) error {
	s.sessMu.RLock()
	dst, ok := s.sess[id]
	s.sessMu.RUnlock()

	if !ok {
		return fmt.Errorf("Client %v not connected", id)
	}

	select {
	case <-dst.ctx.Done():
		return fmt.Errorf("Client %v not connected", id)
	default:
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		case dst.WriteCh <- msg:
		}
	}

	return nil
}
