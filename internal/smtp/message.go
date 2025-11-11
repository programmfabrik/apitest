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

// receivedMessage contains a single email message as received via SMTP.
type receivedMessage struct {
	index          int
	smtpFrom       string
	smtpRcptTo     []string
	rawMessageData []byte
	receivedAt     time.Time
	content        *receivedContent
}

// receivedPart contains a single part of a multipart message as received
// via SMTP.
type receivedPart struct {
	index   int
	content *receivedContent
}

// receivedContent contains the contents of an email message or multipart part.
type receivedContent struct {
	headers           map[string][]string
	body              []byte
	contentType       string
	contentTypeParams map[string]string
	isMultipart       bool
	multiparts        []*receivedPart
}

// contentHaver makes it easier to write algorithms over types that have an
// email message and/or multipart content.
type contentHaver interface {
	Content() *receivedContent
}

// newReceivedMessage parses a raw message as received via SMTP into a
// ReceivedMessage struct.
// Incoming data is truncated after the given maximum message size.
// If a maxMessageSize of 0 is given, this function will default to using
// DefaultMaxMessageSize.
func newReceivedMessage(
	index int,
	from string,
	rcptTo []string,
	rawMessageData []byte,
	receivedAt time.Time,
	maxMessageSize int64,
) (msg *receivedMessage, err error) {

	var (
		parsedMsg *mail.Message
	)

	if maxMessageSize == 0 {
		maxMessageSize = defaultMaxMessageSize
	}

	parsedMsg, err = mail.ReadMessage(io.LimitReader(bytes.NewReader(rawMessageData), maxMessageSize))
	if err != nil {
		return nil, fmt.Errorf("could not parse message: %w", err)
	}

	return newReceivedMessageFromParsed(
		index,
		from,
		rcptTo,
		rawMessageData,
		receivedAt,
		maxMessageSize,
		parsedMsg,
	)
}

// newReceivedMessageFromParsed creates a ReceivedMessage from an already parsed email.
//
// See the documentation of NewReceivedMessage for more details.
func newReceivedMessageFromParsed(
	index int,
	from string,
	rcptTo []string,
	rawMessageData []byte,
	receivedAt time.Time,
	maxMessageSize int64,
	parsedMsg *mail.Message,
) (msg *receivedMessage, err error) {
	var (
		content *receivedContent
	)

	content, err = newReceivedContent(parsedMsg.Header, parsedMsg.Body, maxMessageSize)
	if err != nil {
		return nil, fmt.Errorf("could not parse content: %w", err)
	}

	msg = &receivedMessage{
		index:          index,
		smtpFrom:       from,
		smtpRcptTo:     rcptTo,
		rawMessageData: rawMessageData,
		receivedAt:     receivedAt,
		content:        content,
	}

	return msg, nil
}

// newReceivedPart parses a MIME multipart part into a ReceivedPart struct.
// maxMessageSize is passed through to NewReceivedContent (see its documentation for details).
func newReceivedPart(index int, p *multipart.Part, maxMessageSize int64) (part *receivedPart, err error) {
	var (
		content *receivedContent
	)

	content, err = newReceivedContent(p.Header, p, maxMessageSize)
	if err != nil {
		return nil, fmt.Errorf("could not parse content: %w", err)
	}

	part = &receivedPart{
		index:   index,
		content: content,
	}

	return part, nil
}

// newReceivedContent parses a message or part headers and body into a ReceivedContent struct.
// Incoming data is truncated after the given maximum message size.
// If a maxMessageSize of 0 is given, this function will default to using
// DefaultMaxMessageSize.
func newReceivedContent(
	headers map[string][]string,
	bodyReader io.Reader,
	maxMessageSize int64,
) (content *receivedContent, err error) {
	var (
		body []byte
	)

	if maxMessageSize == 0 {
		maxMessageSize = defaultMaxMessageSize
	}

	headers = preprocessHeaders(headers)

	body, err = io.ReadAll(wrapBodyReader(bodyReader, headers, maxMessageSize))
	if err != nil {
		return nil, fmt.Errorf("could not read body: %w", err)
	}

	content = &receivedContent{
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

			part, err := newReceivedPart(i, rawPart, maxMessageSize)
			if err != nil {
				return nil, fmt.Errorf("could not parse message part: %w", err)
			}

			content.multiparts = append(content.multiparts, part)
		}
	}

	return content, nil
}

// preprocessHeaders decodes header values that were encoded according to RFC2047.
func preprocessHeaders(headers map[string][]string) (out map[string][]string) {
	var (
		decoder mime.WordDecoder
	)

	out = map[string][]string{}

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

func (c *receivedContent) ContentType() (contentType string) {
	return c.contentType
}

func (c *receivedContent) ContentTypeParams() (contentTypeParams map[string]string) {
	return c.contentTypeParams
}

func (c *receivedContent) Body() (body []byte) {
	return c.body
}

func (c *receivedContent) Headers() (headers map[string][]string) {
	return c.headers
}

func (c *receivedContent) IsMultipart() (isMultipart bool) {
	return c.isMultipart
}

func (c *receivedContent) Multiparts() (multiparts []*receivedPart) {
	return c.multiparts
}

func (m *receivedMessage) Content() (content *receivedContent) {
	return m.content
}

func (m *receivedMessage) Index() (index int) {
	return m.index
}

func (m *receivedMessage) RawMessageData() (rawMessageData []byte) {
	return m.rawMessageData
}

func (m *receivedMessage) ReceivedAt() (receivedAt time.Time) {
	return m.receivedAt
}

func (m *receivedMessage) SmtpFrom() (smtpFrom string) {
	return m.smtpFrom
}

func (m *receivedMessage) SmtpRcptTo() (smtpRcptTo []string) {
	return m.smtpRcptTo
}

func (p *receivedPart) Content() (content *receivedContent) {
	return p.content
}

func (p *receivedPart) Index() (index int) {
	return p.index
}
