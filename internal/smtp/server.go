package smtp

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sync"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/sirupsen/logrus"
)

// Server contains a basic SMTP server for testing purposes.
//
// It will accept incoming messages and save them to an internal list of
// received messages, which can be queried using the appropriate methods
// of Server.
type Server struct {
	server           *smtp.Server
	receivedMessages []*ReceivedMessage

	mutex sync.RWMutex
}

type session struct {
	server *Server
	conn   *smtp.Conn

	from   string
	rcptTo []string
}

func NewServer(addr string) *Server {
	server := new(Server)

	backend := smtp.BackendFunc(func(c *smtp.Conn) (smtp.Session, error) {
		return newSession(server, c)
	})

	s := smtp.NewServer(backend)
	s.Addr = addr
	s.EnableSMTPUTF8 = true
	s.EnableBINARYMIME = true

	server.server = s

	return server
}

// ListenAndServe runs the SMTP server. It will not return until the server is
// shut down or otherwise aborts.
func (s *Server) ListenAndServe() error {
	return s.server.ListenAndServe()
}

// Shutdown shuts down the SMTP server that was previously started using
// ListenAndServe.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// ReceivedMessage returns a message that the server has retrieved
// by its index in the list of received messages.
func (s *Server) ReceivedMessage(idx int) (*ReceivedMessage, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	if idx >= len(s.receivedMessages) {
		return nil, fmt.Errorf(
			"Server does not contain message with index %d", idx,
		)
	}

	return s.receivedMessages[idx], nil
}

// ReceivedMessages returns the list of all messages that the server has
// retrieved.
func (s *Server) ReceivedMessages() []*ReceivedMessage {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// We copy the slice to avoid race conditions when the receivedMessages slice is updated.
	// It's just a slice of pointers, so it should be relatively lightweight.
	view := make([]*ReceivedMessage, len(s.receivedMessages))
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

func newSession(server *Server, c *smtp.Conn) (smtp.Session, error) {
	return &session{
		server: server,
		conn:   c,
	}, nil
}

// Implements smtp.Session's Data method.
func (s *session) Data(r io.Reader) error {
	rawData, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("could not read mail data from SMTP: %w", err)
	}

	s.server.mutex.Lock()
	defer s.server.mutex.Unlock()

	now := time.Now()

	logrus.Infof("SMTP: Received message from %s to %v at %v", s.from, s.rcptTo, now)
	msg, err := NewReceivedMessage(s.from, s.rcptTo, rawData, now)
	if err != nil {
		return fmt.Errorf("error constructing ReceivedMessage in SMTP server: %w", err)
	}

	s.server.receivedMessages = append(s.server.receivedMessages, msg)

	return nil
}

// Implements smtp.Session's Logout method.
func (s *session) Logout() error {
	s.Reset()
	return nil
}

// Implements smtp.Session's Mail method.
func (s *session) Mail(from string, opts *smtp.MailOptions) error {
	// opts are currently ignored
	s.from = from
	return nil
}

// Implements smtp.Session's Rcpt method.
func (s *session) Rcpt(to string, opts *smtp.RcptOptions) error {
	// opts are currently ignored
	s.rcptTo = append(s.rcptTo, to)
	return nil
}

// Implements smtp.Session's Reset method.
func (s *session) Reset() {
	s.from = ""
	s.rcptTo = nil
}
