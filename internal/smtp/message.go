package smtp

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net/mail"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// ReceivedMessage contains a single email message as received via SMTP.
type ReceivedMessage struct {
	index int

	smtpFrom       string
	smtpRcptTo     []string
	rawMessageData []byte
	receivedAt     time.Time

	content *ReceivedContent
}

// ReceivedPart contains a single part of a multipart message as received
// via SMTP.
type ReceivedPart struct {
	index int

	content *ReceivedContent
}

// ReceivedContent contains the contents of an email message or multipart part.
type ReceivedContent struct {
	headers map[string][]string
	body    []byte

	contentType       string
	contentTypeParams map[string]string
	isMultipart       bool

	multiparts []*ReceivedPart
}

// ContentHaver makes it easier to write algorithms over types that have an
// email message and/or multipart content.
type ContentHaver interface {
	Content() *ReceivedContent
}

// NewReceivedMessage parses a raw message as received via SMTP into a
// ReceivedMessage struct.
//
// Incoming data is truncated after the given maximum message size.
// If a maxMessageSize of 0 is given, this function will default to using
// DefaultMaxMessageSize.
func NewReceivedMessage(
	index int,
	from string, rcptTo []string, rawMessageData []byte, receivedAt time.Time,
	maxMessageSize int64,
) (*ReceivedMessage, error) {
	if maxMessageSize == 0 {
		maxMessageSize = DefaultMaxMessageSize
	}

	parsedMsg, err := mail.ReadMessage(io.LimitReader(bytes.NewReader(rawMessageData), maxMessageSize))
	if err != nil {
		return nil, fmt.Errorf("could not parse message: %w", err)
	}

	content, err := NewReceivedContent(parsedMsg.Header, parsedMsg.Body, maxMessageSize)
	if err != nil {
		return nil, fmt.Errorf("could not parse content: %w", err)
	}

	msg := &ReceivedMessage{
		index:          index,
		smtpFrom:       from,
		smtpRcptTo:     rcptTo,
		rawMessageData: rawMessageData,
		receivedAt:     receivedAt,
		content:        content,
	}

	return msg, nil
}

// NewReceivedPart parses a MIME multipart part into a ReceivedPart struct.
//
// maxMessageSize is passed through to NewReceivedContent (see its documentation for details).
func NewReceivedPart(index int, p *multipart.Part, maxMessageSize int64) (*ReceivedPart, error) {
	content, err := NewReceivedContent(p.Header, p, maxMessageSize)
	if err != nil {
		return nil, fmt.Errorf("could not parse content: %w", err)
	}

	part := &ReceivedPart{
		index:   index,
		content: content,
	}

	return part, nil
}

// NewReceivedContent parses a message or part headers and body into a ReceivedContent struct.
//
// Incoming data is truncated after the given maximum message size.
// If a maxMessageSize of 0 is given, this function will default to using
// DefaultMaxMessageSize.
func NewReceivedContent(
	headers map[string][]string, bodyReader io.Reader, maxMessageSize int64,
) (*ReceivedContent, error) {
	if maxMessageSize == 0 {
		maxMessageSize = DefaultMaxMessageSize
	}

	headers = preprocessHeaders(headers)

	body, err := io.ReadAll(wrapBodyReader(bodyReader, headers, maxMessageSize))
	if err != nil {
		return nil, fmt.Errorf("could not read body: %w", err)
	}

	content := &ReceivedContent{
		headers: headers,
		body:    body,
	}

	rawContentType, ok := headers["Content-Type"]
	if ok && rawContentType[0] != "" && len(rawContentType) > 0 {
		content.contentType, content.contentTypeParams, err = mime.ParseMediaType(rawContentType[0])
		if err != nil {
			return nil, fmt.Errorf("could not parse Content-Type: %w", err)
		}

		// case-sensitive comparison of the content type is permitted here,
		// since mime.ParseMediaType is documented to return the media type
		// in lower case.
		content.isMultipart = strings.HasPrefix(content.contentType, "multipart/")
	}

	if content.IsMultipart() {
		boundary, ok := content.contentTypeParams["boundary"]
		if !ok {
			return nil, fmt.Errorf("encountered multipart message without defined boundary")
		}

		r := multipart.NewReader(bytes.NewReader(content.body), boundary)

		for i := 0; ; i++ {
			rawPart, err := r.NextRawPart()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				} else {
					return nil, fmt.Errorf("could not read multipart: %w", err)
				}
			}

			part, err := NewReceivedPart(i, rawPart, maxMessageSize)
			if err != nil {
				return nil, fmt.Errorf("could not parse message part: %w", err)
			}

			content.multiparts = append(content.multiparts, part)
		}
	}

	return content, nil
}

// preprocessHeaders decodes header values that were encoded according to RFC2047.
func preprocessHeaders(headers map[string][]string) map[string][]string {
	var decoder mime.WordDecoder

	out := make(map[string][]string)

	for k, vs := range headers {
		out[k] = make([]string, len(vs))

		for i := range vs {
			dec, err := decoder.DecodeHeader(vs[i])
			if err != nil {
				logrus.Warn("could not decode Q-Encoding in header:", err)
			} else {
				out[k][i] = dec
			}
		}
	}

	return out
}

// wrapBodyReader wraps the reader for a message / part body with size
// limitation and quoted-printable/base64 decoding (the latter based on
// the Content-Transfer-Encoding header, if any is set).
func wrapBodyReader(r io.Reader, headers map[string][]string, maxMessageSize int64) io.Reader {
	r = io.LimitReader(r, maxMessageSize)

	enc, ok := headers["Content-Transfer-Encoding"]
	if ok {
		if len(enc) != 1 {
			logrus.Error("Content-Transfer-Encoding must have exactly one value")
		}

		switch enc[0] {
		case "base64":
			r = base64.NewDecoder(base64.StdEncoding, r)
		case "quoted-printable":
			r = quotedprintable.NewReader(r)
		default:
			logrus.Errorf("encountered unknown Content-Transfer-Encoding %q", enc)
		}
	}

	return r
}

// =======
// Getters
// =======

func (c *ReceivedContent) ContentType() string {
	return c.contentType
}

func (c *ReceivedContent) ContentTypeParams() map[string]string {
	return c.contentTypeParams
}

func (c *ReceivedContent) Body() []byte {
	return c.body
}

func (c *ReceivedContent) Headers() map[string][]string {
	return c.headers
}

func (c *ReceivedContent) IsMultipart() bool {
	return c.isMultipart
}

func (c *ReceivedContent) Multiparts() []*ReceivedPart {
	return c.multiparts
}

func (m *ReceivedMessage) Content() *ReceivedContent {
	return m.content
}

func (m *ReceivedMessage) Index() int {
	return m.index
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

func (p *ReceivedPart) Content() *ReceivedContent {
	return p.content
}

func (p *ReceivedPart) Index() int {
	return p.index
}
