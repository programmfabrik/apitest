package smtp

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/mail"
	"net/textproto"
	"strings"
	"time"
)

// ReceivedMessage contains a single email message as received via SMTP.
type ReceivedMessage struct {
	smtpFrom       string
	smtpRcptTo     []string
	rawMessageData []byte
	receivedAt     time.Time

	headers mail.Header
	body    []byte

	contentType       string
	contentTypeParams map[string]string

	isMultipart bool
	multiparts  []*ReceivedPart
}

// ReceivedPart contains a single part of a multipart message as received
// via SMTP.
type ReceivedPart struct {
	headers textproto.MIMEHeader
	body    []byte
}

func NewReceivedMessage(
	from string, rcptTo []string, rawMessageData []byte, receivedAt time.Time,
) (*ReceivedMessage, error) {
	parsedMsg, err := mail.ReadMessage(bytes.NewReader(rawMessageData))
	if err != nil {
		return nil, fmt.Errorf("could not parse message: %w", err)
	}

	// TODO: Limit length?
	body, err := io.ReadAll(parsedMsg.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read message body: %w", err)
	}

	msg := &ReceivedMessage{
		smtpFrom:       from,
		smtpRcptTo:     rcptTo,
		rawMessageData: rawMessageData,
		receivedAt:     receivedAt,
		headers:        parsedMsg.Header,
		body:           body,
	}

	rawContentType := msg.headers.Get("Content-Type")
	if rawContentType != "" {
		msg.contentType, msg.contentTypeParams, err = mime.ParseMediaType(msg.contentType)
		if err != nil {
			return nil, fmt.Errorf("could not parse Content-Type: %w", err)
		}

		// case-sensitive comparison of the content type is permitted here,
		// since mime.ParseMediaType is documented to return the media type
		// in lower case.
		msg.isMultipart = strings.HasPrefix(msg.contentType, "multipart/")
	}

	if msg.isMultipart {
		boundary, ok := msg.contentTypeParams["boundary"]
		if !ok {
			return nil, fmt.Errorf("encountered multipart message without defined boundary")
		}

		r := multipart.NewReader(bytes.NewReader(msg.body), boundary)

		for {
			rawPart, err := r.NextPart()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				} else {
					return nil, fmt.Errorf("could not read multipart: %w", err)
				}
			}

			part, err := NewReceivedPart(rawPart)
			if err != nil {
				return nil, fmt.Errorf("could not parse message part: %w", err)
			}

			msg.multiparts = append(msg.multiparts, part)
		}
	}

	return msg, nil
}

func NewReceivedPart(p *multipart.Part) (*ReceivedPart, error) {
	// TODO: Limit length?
	body, err := io.ReadAll(p)
	if err != nil {
		return nil, fmt.Errorf("could not read message part body: %w", err)
	}

	part := &ReceivedPart{
		headers: p.Header,
		body:    body,
	}

	return part, nil
}

// =======
// Getters
// =======

func (m *ReceivedMessage) ContentType() string {
	return m.contentType
}

func (m *ReceivedMessage) Body() []byte {
	return m.body
}

func (m *ReceivedMessage) Headers() mail.Header {
	return m.headers
}

func (m *ReceivedMessage) IsMultipart() bool {
	return m.isMultipart
}

func (m *ReceivedMessage) Multiparts() []*ReceivedPart {
	return m.multiparts
}

func (m *ReceivedMessage) RawMessageData() []byte {
	return m.rawMessageData
}

func (m *ReceivedMessage) ReceivedAt() time.Time {
	return m.receivedAt
}

func (m *ReceivedMessage) SmtpFrom() string {
	return m.smtpFrom
}

func (m *ReceivedMessage) SmtpRcptTo() []string {
	return m.smtpRcptTo
}

func (p *ReceivedPart) Body() []byte {
	return p.body
}

func (p *ReceivedPart) Headers() textproto.MIMEHeader {
	return p.headers
}
