package server_test

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/disq/msghub"
	"github.com/disq/msghub/server"
)

type base struct {
	t      *testing.T
	ctx    context.Context
	cancel context.CancelFunc

	srv *server.Server
}

// Printf implements the msghub.Logger interface for base
func (b *base) Printf(fmt string, a ...interface{}) {
	b.t.Logf("[server log] "+fmt, a...)
}

func setup(t *testing.T) *base {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())

	b := &base{
		t:      t,
		ctx:    ctx,
		cancel: cancel,
	}
	b.srv = server.New(ctx, b)
	return b
}

// Teardown shuts the server down by cancelling its parent context
func (b *base) Teardown() {
	b.cancel()
}

type client struct {
	srv  *base
	conn net.Conn
	rd   *bufio.Reader
	wr   *bufio.Writer

	name string
}

const clientReadTimeout = 1 * time.Second

// NewClient creates and connects a new client to server
func (b *base) NewClient(name string) *client {
	serverSock, clientSock := net.Pipe()

	c := client{
		srv:  b,
		conn: clientSock,
		rd:   bufio.NewReader(clientSock),
		wr:   bufio.NewWriter(clientSock),

		name: name,
	}

	go b.srv.HandleConn(serverSock)

	return &c
}

// Printf implements the msghub.Logger interface for client
func (c *client) Printf(fmt string, a ...interface{}) {
	args := append([]interface{}{}, c.name)
	args = append(args, a...)

	c.srv.t.Logf("[client:%v] "+fmt, args...)
}

// Send sends a message up to the server
func (c *client) Send(s string) {
	c.wr.WriteString(s + "\n")
	c.wr.Flush()
	c.srv.t.Logf("[client:%v] Sent: %q", c.name, s)
}

// ReadFlush reads all data for clientReadTimeout and discards it
func (c *client) ReadFlush() error {
	c.conn.SetReadDeadline(time.Now().Add(clientReadTimeout))

	for {
		_, err := c.rd.ReadString('\n')
		if err != nil && msghub.IsTimeoutError(err) {
			// deadline reached
			return nil
		}
		if err != nil {
			return err
		}
		//c.Printf("Read: %q", ln)
	}
}

// Expect reads a single line (for clientReadTimeout) and compares it with want
func (c *client) Expect(want string) error {
	c.conn.SetReadDeadline(time.Now().Add(clientReadTimeout))

	got, err := c.rd.ReadString('\n')
	if err != nil {
		return err
	}
	got = strings.TrimRight(got, "\n")
	c.Printf("Read: %q", got)
	if got != want {
		return fmt.Errorf("Got %q, want %q", got, want)
	}

	return nil
}

// TestCommandsOneClient is the single-client test
func TestCommandsOneClient(t *testing.T) {
	base := setup(t)
	c := base.NewClient("alice")

	var tdata = []struct {
		input    string
		expected string
	}{
		{"a", "1"},
		{"w", "No other clients seem to have connected."},
		{"s 1 test", "Error in command: Can't send messages to yourself!"},
	}

	flushClient(c)

	for _, tc := range tdata {

		c.Send(tc.input)
		if err := c.Expect(tc.expected); err != nil {
			t.Errorf("Handler(%q): %v", tc.input, err)
		}
	}

	base.Teardown()
}

type multiClientTestCase struct {
	sendClient int
	payload    string

	receiveClients []int
	expected       []string
}

// TestCommandsChat is the multi-client test
func TestCommandsChat(t *testing.T) {
	base := setup(t)

	c1 := base.NewClient("alice")
	time.Sleep(100 * time.Millisecond) // make sure we have the right order
	c2 := base.NewClient("bob")
	time.Sleep(100 * time.Millisecond)
	c3 := base.NewClient("charlie")

	clients := []*client{c1, c2, c3}

	flushClient(clients...)

	var tdata = []multiClientTestCase{
		{0, "a", []int{0}, []string{"1"}},
		{1, "a", []int{1}, []string{"2"}},
		{1, "w", []int{1}, []string{"List of clients: 1,3"}},
		{0, "w", []int{0}, []string{"List of clients: 2,3"}},
		{0, "s 2 test", []int{0, 1}, []string{"Sent to 1 client", "Message from 1: test"}},
		{1, "s 1,3 test...", []int{1, 0, 2}, []string{"Sent to 2 clients", "Message from 2: test...", "Message from 2: test..."}},
	}

	for _, tc := range tdata {
		clients[tc.sendClient].Send(tc.payload)

		for idx, rcv := range tc.receiveClients {
			if err := clients[rcv].Expect(tc.expected[idx]); err != nil {
				t.Errorf("Handler(%v->%v:%q): %v", tc.sendClient, rcv, tc.payload, err)
			}
		}
	}

	base.Teardown()
}

func flushClient(clients ...*client) {
	var wg sync.WaitGroup

	for _, c := range clients {
		c := c

		wg.Add(1)
		go func() {
			c.ReadFlush()
			wg.Done()
		}()
	}
	wg.Wait()
}
