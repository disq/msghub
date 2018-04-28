package main

import "fmt"

// ClientMessage is a message sent from client-to-client
type ClientMessage struct {
	from     uint64
	contents string
}

// SystemMessage is a global system announcement
type SystemMessage struct {
	contents string
}

// CommandMessage is a message response
type CommandMessage struct {
	contents string
}

type Message interface {
	Read() string
}

func (m ClientMessage) Read() string {
	return fmt.Sprintf("Message from %v: %v", m.from, m.contents)
}

func (m SystemMessage) Read() string {
	return fmt.Sprintf("System message: %v", m.contents)
}

func (m CommandMessage) Read() string {
	return fmt.Sprintf("%v", m.contents)
}
