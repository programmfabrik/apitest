package smtp

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/emersion/go-smtp"
	"github.com/sirupsen/logrus"
)

const defaultMaxMessageSize = 30 * 1024 * 1024 // 30MiB

// Server contains a basic SMTP server for testing purposes.
// It will accept incoming messages and save them to an internal list of
// received messages, which can be queried using the appropriate methods
// of Server.
type Server struct {
	server           *smtp.Server
	receivedMessages []*ReceivedMessage
	maxMessageSize   int64
	mutex            sync.RWMutex
	clock            func() time.Time // making clock mockable for unit testing
}

type session struct {
	server *Server
	conn   *smtp.Conn
	from   string
	rcptTo []string
}

// NewServer creates a new testing SMTP server.
// The new server will listen at the provided address.
// Incoming messages are truncated after the given maximum message size.
// If a maxMessageSize of 0 is given, Server will default to using
// DefaultMaxMessageSize.
func NewServer(addr string, maxMessageSize int64) (server *Server) {
	var (
		s *smtp.Server
	)

	if maxMessageSize == 0 {
		maxMessageSize = defaultMaxMessageSize
	}

	server = &Server{
		maxMessageSize: maxMessageSize,
		clock:          time.Now,
	}

	backend := smtp.BackendFunc(func(c *smtp.Conn) (sess smtp.Session, err error) {
		return newSession(server, c)
	})

	s = smtp.NewServer(backend)
	s.Addr = addr
	s.EnableSMTPUTF8 = true
	s.EnableBINARYMIME = true

	server.server = s

	return server
}

// AppendMessage adds a custom message to the Server's storage.
// The index of the provided message will be updated to the index at which
// it was actually inserted into the Server's storage.
func (s *Server) AppendMessage(msg *ReceivedMessage) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	msg.index = len(s.receivedMessages)
	s.receivedMessages = append(s.receivedMessages, msg)
}

// ListenAndServe runs the SMTP server. It will not return until the server is
// shut down or otherwise aborts.
func (s *Server) ListenAndServe() (err error) {
	return s.server.ListenAndServe()
}

// Shutdown shuts down the SMTP server that was previously started using
// ListenAndServe.
func (s *Server) Shutdown(ctx context.Context) (err error) {
	return s.server.Shutdown(ctx)
}

// ReceivedMessage returns a message that the server has retrieved
// by its index in the list of received messages.
func (s *Server) ReceivedMessage(idx int) (msg *ReceivedMessage, err error) {
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
func (s *Server) ReceivedMessages() (msgs []*ReceivedMessage) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// We copy the slice to avoid race conditions when the receivedMessages slice is updated.
	// It's just a slice of pointers, so it should be relatively lightweight.
	view := make([]*ReceivedMessage, len(s.receivedMessages))
	copy(view, s.receivedMessages)

	return view
}

func newSession(server *Server, c *smtp.Conn) (sess smtp.Session, err error) {
	return &session{
		server: server,
		conn:   c,
	}, nil
}

// Implements smtp.Session's Data method.
func (s *session) Data(r io.Reader) (err error) {
	rawData, err := io.ReadAll(io.LimitReader(r, s.server.maxMessageSize))
	if err != nil {
		return fmt.Errorf("could not read mail data from SMTP: %w", err)
	}

	s.server.mutex.Lock()
	defer s.server.mutex.Unlock()

	idx := len(s.server.receivedMessages)
	now := s.server.clock()

	msg, err := NewReceivedMessage(
		idx, s.from, s.rcptTo, rawData, now, s.server.maxMessageSize,
	)
	if err != nil {
		errWrapped := fmt.Errorf("constructing ReceivedMessage in SMTP server: %w", err)
		logrus.Error("SMTP:", errWrapped) // this is logged in our server
		return errWrapped                 // this is returned via SMTP
	}
	logrus.Infof("smtp: From: %q To: %v", s.from, s.rcptTo)

	s.server.receivedMessages = append(s.server.receivedMessages, msg)

	return nil
}

// Implements smtp.Session's Logout method.
func (s *session) Logout() (err error) {
	s.Reset()
	return nil
}

// Implements smtp.Session's Mail method.
func (s *session) Mail(from string, opts *smtp.MailOptions) (err error) {
	// opts are currently ignored
	s.from = from
	return nil
}

// Implements smtp.Session's Rcpt method.
func (s *session) Rcpt(to string, opts *smtp.RcptOptions) (err error) {
	// opts are currently ignored
	s.rcptTo = append(s.rcptTo, to)
	return nil
}

// Implements smtp.Session's Reset method.
func (s *session) Reset() {
	s.from = ""
	s.rcptTo = nil
}
