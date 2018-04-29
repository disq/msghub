package server

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

// ShutdownMessage is a global system announcement after which the connection is terminated
type ShutdownMessage struct {
	*SystemMessage
}

// CommandMessage is a message response
type CommandMessage struct {
	contents string
}

// Message is the interface for messages
type Message interface {
	Read() string
}

// Read implements the Message interface for ClientMessage
func (m ClientMessage) Read() string {
	return fmt.Sprintf("Message from %v: %v", m.from, m.contents)
}

// Read implements the Message interface for SystemMessage
func (m SystemMessage) Read() string {
	return fmt.Sprintf("System message: %v", m.contents)
}

// Read implements the Message interface for CommandMessage
func (m CommandMessage) Read() string {
	return fmt.Sprintf("%v", m.contents)
}
