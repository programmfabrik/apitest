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
	"regexp"
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

// NewReceivedMessage parses a raw message as received via SMTP into a
// ReceivedMessage struct.
//
// Incoming data is truncated after the given maximum message size.
// If a maxMessageSize of 0 is given, this function will default to using
// DefaultMaxMessageSize.
func NewReceivedMessage(
	from string, rcptTo []string, rawMessageData []byte, receivedAt time.Time,
	maxMessageSize int64,
) (*ReceivedMessage, error) {
	if maxMessageSize == 0 {
		maxMessageSize = DefaultMaxMessageSize
	}

	parsedMsg, err := mail.ReadMessage(bytes.NewReader(rawMessageData))
	if err != nil {
		return nil, fmt.Errorf("could not parse message: %w", err)
	}

	body, err := io.ReadAll(io.LimitReader(parsedMsg.Body, maxMessageSize))
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
		msg.contentType, msg.contentTypeParams, err = mime.ParseMediaType(rawContentType)
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

			part, err := NewReceivedPart(rawPart, maxMessageSize)
			if err != nil {
				return nil, fmt.Errorf("could not parse message part: %w", err)
			}

			msg.multiparts = append(msg.multiparts, part)
		}
	}

	return msg, nil
}

// SearchPartsByHeader returns the list of all received multiparts that
// have at least one header matching the given regular expression.
//
// For details on how the matching is performed, please refer to the
// documentation for Server.SearchByHeader.
//
// If the message is not a multipart message, this returns nil.
// If no matching multiparts are found, this may return nil or an empty
// list.
func (m *ReceivedMessage) SearchPartsByHeader(re *regexp.Regexp) []*ReceivedPart {
	if !m.IsMultipart() {
		return nil
	}

	multiparts := m.Multiparts()

	headerIdxList := make([]map[string][]string, len(multiparts))
	for i, v := range multiparts {
		headerIdxList[i] = v.Headers()
	}

	foundIndices := searchByHeaderCommon(headerIdxList, re)

	results := make([]*ReceivedPart, 0, len(foundIndices))
	for _, idx := range foundIndices {
		results = append(results, multiparts[idx])
	}

	return results
}

// NewReceivedPart parses a MIME multipart part into a ReceivedPart struct.
//
// Incoming data is truncated after the given maximum message size.
// If a maxMessageSize of 0 is given, this function will default to using
// DefaultMaxMessageSize.
func NewReceivedPart(p *multipart.Part, maxMessageSize int64) (*ReceivedPart, error) {
	if maxMessageSize == 0 {
		maxMessageSize = DefaultMaxMessageSize
	}

	body, err := io.ReadAll(io.LimitReader(p, maxMessageSize))
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
