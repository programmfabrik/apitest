package smtp

import (
	"fmt"
	"regexp"
	"sync"

	"github.com/emersion/go-smtp"
)

// Server contains a basic SMTP server for testing purposes.
//
// It will accept incoming messages and save them to an internal list of
// received messages, which can be queried using the appropriate methods
// of Server.
type Server struct {
	server           smtp.Server
	receivedMessages []*ReceivedMessage

	mutex sync.RWMutex
}

type session struct {
	// TODO
}

func NewServer(addr string) *Server {
	server := &Server{
		receivedMessages: make([]receivedMessage),
	}

	backend := panic("not implemented") // TODO

	server.server = smtp.NewServer(backend)
	server.Addr = addr
	// TODO: Enable SMTPUTF8?
	// TODO: Enable BINARYMIME?

	return server
}

// ListenAndServe runs the SMTP server. It will not return until the server is
// shut down or otherwise aborts.
func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

// Shutdown shuts down the SMTP server that was previously started using
// ListenAndServe.
func (s *Server) Shutdown() error {
	return s.server.Shutdown()
}

// ReceivedMessage returns a message that the server has retrieved
// by its index in the list of received messages.
func (s *Server) ReceivedMessage(idx int) (*ReceivedMessage, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	msg, ok := s.receivedMessages[idx]
	if !ok {
		return nil, fmt.Errorf(
			"Server does not contain message with index %d", idx,
		)
	}

	return msg, nil
}

// ReceivedMessages returns the list of all messages that the server has
// retrieved.
func (s *Server) ReceivedMessages() []ReceivedMessage {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// We copy the slice to avoid race conditions when the receivedMessages slice is updated.
	// It's just a slice of pointers, so it should be relatively lightweight.
	view := make([]*ReceivedMessage, len(s.receivedMessage))
	copy(view, s.receivedMessages)

	return view
}

// SearchByHeader returns the list of all received messages that have at
// least one header matching the given regular expression.
func (s *Server) SearchByHeader(re *regexp.Regexp) []ReceivedMessage {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// TODO
	panic("not implemented")
}
