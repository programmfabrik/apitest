package smtp

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"net/mail"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/programmfabrik/apitest/internal/handlerutil"
	"github.com/programmfabrik/golib"
	"github.com/sirupsen/logrus"
)

//go:embed gui_index.html
var guiIndexTemplateSrc string
var guiIndexTemplate = template.Must(template.New("gui_index").Parse(guiIndexTemplateSrc))

//go:embed gui_message.html
var guiMessageTemplateSrc string
var guiMessageTemplate = template.Must(template.
	New("gui_message").
	Funcs(sprig.TxtFuncMap()).
	Parse(guiMessageTemplateSrc),
)

type smtpHTTPHandler struct {
	server *Server
	prefix string
}

// RegisterRoutes sets up HTTP routes for inspecting the SMTP Server's
// received messages.
func (s *Server) RegisterRoutes(mux *http.ServeMux, prefix string, skipLogs bool) {
	handler := &smtpHTTPHandler{
		server: s,
		prefix: path.Join(prefix, "smtp"),
	}

	mux.Handle(handler.prefix, handlerutil.LogH(skipLogs, handler))
	mux.Handle(handler.prefix+"/", handlerutil.LogH(skipLogs, handler))
}

func (h *smtpHTTPHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := path.Clean(r.URL.Path)
	path = strings.TrimPrefix(path, h.prefix)
	path = strings.TrimPrefix(path, "/")

	if path == "" {
		h.handleMessageIndex(w, r)
		return
	}

	pathParts := strings.Split(path, "/")
	fmt.Println(pathParts)

	// We now know that pathParts must have at least length 1, since empty path
	// was already handled above.

	switch pathParts[0] {
	case "gui":
		h.routeGUIEndpoint(w, pathParts)
	case "postmessage":
		h.handlePostMessage(w, r)
	default:
		h.routeMessageEndpoint(w, r, pathParts)
	}
}

func (h *smtpHTTPHandler) routeGUIEndpoint(w http.ResponseWriter, pathParts []string) {
	if len(pathParts) == 0 {
		handlerutil.RespondWithErr(
			w, http.StatusInternalServerError,
			fmt.Errorf("routeGUIEndpoint was called with empty pathParts"),
		)
		return
	}

	if len(pathParts) == 1 {
		h.handleGUIIndex(w)
		return
	}

	// We know at this point that len(pathParts) must be >= 2

	msg, ok := h.retrieveMessage(w, pathParts[1])
	if !ok {
		return
	}

	if len(pathParts) == 2 {
		h.handleGUIMessage(w, msg)
		return
	}

	// If routing failed, return status 404.
	w.WriteHeader(http.StatusNotFound)
}

func (h *smtpHTTPHandler) routeMessageEndpoint(w http.ResponseWriter, r *http.Request, pathParts []string) {
	if len(pathParts) == 0 {
		handlerutil.RespondWithErr(
			w, http.StatusInternalServerError,
			fmt.Errorf("routeMessageEndpoint was called with empty pathParts"),
		)
		return
	}

	msg, ok := h.retrieveMessage(w, pathParts[0])
	if !ok {
		return
	}

	if len(pathParts) == 1 {
		h.handleMessageMeta(w, msg)
		return
	}
	if len(pathParts) == 2 && pathParts[1] == "raw" {
		h.handleMessageRaw(w, msg)
		return
	}

	h.subrouteContentEndpoint(w, r, msg.Content(), pathParts[1:])
}

// subrouteContentEndpoint recursively finds a route for the remaining path parts
// based on the given ReceivedContent.
func (h *smtpHTTPHandler) subrouteContentEndpoint(w http.ResponseWriter, r *http.Request, c *ReceivedContent, remainingPathParts []string) {
	ensureIsMultipart := func() bool {
		if !c.IsMultipart() {
			handlerutil.RespondWithErr(w, http.StatusNotFound, fmt.Errorf(
				"multipart endpoint was requested for non-multipart content",
			))
			return false
		}

		return true
	}

	if len(remainingPathParts) == 1 {
		switch remainingPathParts[0] {
		case "body":
			h.handleContentBody(w, c)
			return
		case "multipart":
			if !ensureIsMultipart() {
				return
			}

			h.handleMultipartIndex(w, r, c)
			return
		}
	}

	if len(remainingPathParts) > 1 && remainingPathParts[0] == "multipart" {
		if !ensureIsMultipart() {
			return
		}

		multiparts := c.Multiparts()

		partIdx, err := strconv.Atoi(remainingPathParts[1])
		if err != nil {
			handlerutil.RespondWithErr(
				w, http.StatusBadRequest,
				fmt.Errorf("could not parse multipart index: %w", err),
			)
			return
		}

		if partIdx >= len(multiparts) {
			handlerutil.RespondWithErr(w, http.StatusNotFound, fmt.Errorf(
				"ReceivedContent does not contain multipart with index %d", partIdx,
			))
			return
		}

		part := multiparts[partIdx]

		if len(remainingPathParts) == 2 {
			h.handleMultipartMeta(w, part)
			return
		}

		h.subrouteContentEndpoint(w, r, part.Content(), remainingPathParts[2:])
		return
	}

	// If routing failed, return status 404.
	w.WriteHeader(http.StatusNotFound)
}

func (h *smtpHTTPHandler) handleContentBody(w http.ResponseWriter, c *ReceivedContent) {
	contentType, ok := c.Headers()["Content-Type"]
	if ok {
		w.Header()["Content-Type"] = contentType
	}

	w.Write(c.Body())
}

func (h *smtpHTTPHandler) handleGUIIndex(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err := guiIndexTemplate.Execute(w, map[string]any{"prefix": h.prefix})
	if err != nil {
		logrus.Error("error rendering GUI Index:", err)
	}
}

func (h *smtpHTTPHandler) handleGUIMessage(w http.ResponseWriter, msg *ReceivedMessage) {
	metadata := buildMessageFullMeta(msg)
	metadataJson := golib.JsonStringIndent(metadata, "", "  ")

	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	err := guiMessageTemplate.Execute(w, map[string]any{
		"prefix":       h.prefix,
		"metadata":     metadata,
		"metadataJson": metadataJson,
	})
	if err != nil {
		logrus.Error("error rendering GUI Message:", err)
	}
}

func (h *smtpHTTPHandler) handleMessageIndex(w http.ResponseWriter, r *http.Request) {
	headerSearchRxs, err := extractSearchRegexes(r.URL.Query(), "header")
	if err != nil {
		handlerutil.RespondWithErr(w, http.StatusBadRequest, err)
		return
	}

	receivedMessages := h.server.ReceivedMessages()
	if len(headerSearchRxs) > 0 {
		receivedMessages = SearchByHeader(receivedMessages, headerSearchRxs...)
	}

	messagesOut := make([]any, 0)

	for _, msg := range receivedMessages {
		messagesOut = append(messagesOut, buildMessageBasicMeta(msg))
	}

	out := make(map[string]any)
	out["count"] = len(receivedMessages)
	out["messages"] = messagesOut

	handlerutil.RespondWithJSON(w, http.StatusOK, out)
}

func (h *smtpHTTPHandler) handleMessageMeta(w http.ResponseWriter, msg *ReceivedMessage) {
	handlerutil.RespondWithJSON(w, http.StatusOK, buildMessageFullMeta(msg))
}

func (h *smtpHTTPHandler) handleMessageRaw(w http.ResponseWriter, msg *ReceivedMessage) {
	w.Header().Set("Content-Type", "message/rfc822")
	w.Write(msg.RawMessageData())
}

func (h *smtpHTTPHandler) handleMultipartIndex(w http.ResponseWriter, r *http.Request, c *ReceivedContent) {
	headerSearchRxs, err := extractSearchRegexes(r.URL.Query(), "header")
	if err != nil {
		handlerutil.RespondWithErr(w, http.StatusBadRequest, err)
		return
	}

	multiparts := c.Multiparts()
	if len(headerSearchRxs) > 0 {
		multiparts = SearchByHeader(multiparts, headerSearchRxs...)
	}

	handlerutil.RespondWithJSON(w, http.StatusOK, buildMultipartIndex(multiparts))
}

func (h *smtpHTTPHandler) handleMultipartMeta(w http.ResponseWriter, part *ReceivedPart) {
	handlerutil.RespondWithJSON(w, http.StatusOK, buildMultipartMeta(part))
}

func (h *smtpHTTPHandler) handlePostMessage(w http.ResponseWriter, r *http.Request) {
	maxMessageSize := h.server.maxMessageSize
	if maxMessageSize == 0 {
		maxMessageSize = DefaultMaxMessageSize
	}

	if r.Method != http.MethodPost {
		handlerutil.RespondWithErr(
			w, http.StatusMethodNotAllowed,
			fmt.Errorf("postmessage only accepts POST requests"),
		)
		return
	}

	rawMessageData, err := io.ReadAll(io.LimitReader(r.Body, maxMessageSize))
	if err != nil {
		handlerutil.RespondWithErr(
			w, http.StatusBadRequest,
			fmt.Errorf("reading body: %w", err),
		)
		return
	}

	// Ensure line endings are CRLF
	rawMessageData = bytes.ReplaceAll(rawMessageData, []byte("\r\n"), []byte("\n"))
	rawMessageData = bytes.ReplaceAll(rawMessageData, []byte("\n"), []byte("\r\n"))

	parsedRfcMsg, err := mail.ReadMessage(bytes.NewReader(rawMessageData))
	if err != nil {
		handlerutil.RespondWithErr(
			w, http.StatusBadRequest,
			fmt.Errorf("postmessage could not parse message: %w", err),
		)
		return
	}

	from := parsedRfcMsg.Header.Get("From")
	rcptTo := parsedRfcMsg.Header["To"]
	receivedAt := time.Now()

	msg, err := NewReceivedMessageFromParsed(
		0, // the index will be overriden by Server.AppendMessage below
		from, rcptTo, rawMessageData, receivedAt, maxMessageSize,
		parsedRfcMsg,
	)
	if err != nil {
		handlerutil.RespondWithErr(
			w, http.StatusBadRequest,
			fmt.Errorf("postmessage could not build ReceivedMessage: %w", err),
		)
		return
	}

	h.server.AppendMessage(msg)

	w.WriteHeader(http.StatusOK)
}

// retrieveMessage attempts to retrieve the message referenced by the given index (still in string
// form at this point). If the index could not be read or the message could not be retrieved,
// an according error message will be returned via HTTP.
func (h *smtpHTTPHandler) retrieveMessage(w http.ResponseWriter, sIdx string) (*ReceivedMessage, bool) {
	idx, err := strconv.Atoi(sIdx)
	if err != nil {
		handlerutil.RespondWithErr(
			w, http.StatusBadRequest,
			fmt.Errorf("could not parse message index: %w", err),
		)
		return nil, false
	}

	msg, err := h.server.ReceivedMessage(idx)
	if err != nil {
		handlerutil.RespondWithErr(w, http.StatusNotFound, err)
		return nil, false
	}

	return msg, true
}

func buildContentMeta(c *ReceivedContent) map[string]any {
	contentTypeParams := c.ContentTypeParams()
	if contentTypeParams == nil {
		// Returning an empty map instead of null more closely resembles the semantics
		// of contentTypeParams.
		contentTypeParams = make(map[string]string)
	}

	out := map[string]any{
		"bodySize":          len(c.Body()),
		"isMultipart":       c.IsMultipart(),
		"contentType":       c.ContentType(),
		"contentTypeParams": contentTypeParams,
	}

	headers := make(map[string]any)
	for k, v := range c.Headers() {
		headers[k] = v
	}
	out["headers"] = headers

	if c.IsMultipart() {
		multipartIndex := buildMultipartIndex(c.Multiparts())
		for k, v := range multipartIndex {
			out[k] = v
		}
	}

	return out
}

func buildMessageBasicMeta(msg *ReceivedMessage) map[string]any {
	content := msg.Content()

	out := map[string]any{
		"idx":         msg.Index(),
		"isMultipart": content.IsMultipart(),
		"receivedAt":  msg.ReceivedAt(),
		"smtpFrom":    msg.SmtpFrom(),
		"smtpRcptTo":  msg.SmtpRcptTo(),
	}

	from, ok := content.Headers()["From"]
	if ok {
		out["from"] = from
	}

	to, ok := content.Headers()["To"]
	if ok {
		out["to"] = to
	}

	subject, ok := content.Headers()["Subject"]
	if ok && len(subject) == 1 {
		out["subject"] = subject[0]
	}

	return out
}

func buildMessageFullMeta(msg *ReceivedMessage) map[string]any {
	out := buildMessageBasicMeta(msg)
	contentMeta := buildContentMeta(msg.Content())

	for k, v := range contentMeta {
		out[k] = v
	}

	return out
}

func buildMultipartIndex(parts []*ReceivedPart) map[string]any {
	multipartsOut := make([]any, len(parts))

	for i, part := range parts {
		multipartsOut[i] = buildMultipartMeta(part)
	}

	out := make(map[string]any)
	out["multipartsCount"] = len(parts)
	out["multiparts"] = multipartsOut

	return out
}

func buildMultipartMeta(part *ReceivedPart) map[string]any {
	out := map[string]any{
		"idx": part.Index(),
	}

	contentMeta := buildContentMeta(part.Content())

	for k, v := range contentMeta {
		out[k] = v
	}

	return out
}

// extractSearchRegexes tries to extract the regular expression(s) from the
// referenced query parameter. If no query parameter is given and otherwise
// no error has occurred, this function returns no error.
func extractSearchRegexes(qp url.Values, paramName string) (rgs []*regexp.Regexp, err error) {
	if !qp.Has(paramName) {
		return nil, nil
	}
	defer func() {
		if err == nil {
			println(fmt.Sprintf("%v", rgs))
		}
	}()

	sp := []string{}
	for _, v := range qp[paramName] {
		var searchParams []string
		err := json.Unmarshal([]byte(v), &searchParams)
		if err == nil {
			sp = append(sp, searchParams...)
		} else {
			// this is not a JSON string array, assume string
			sp = append(sp, v)
		}
	}

	for _, p := range sp {
		re, err := regexp.Compile(p)
		if err != nil {
			return nil, fmt.Errorf("could not compile %q regex %q: %w", paramName, p, err)
		}
		rgs = append(rgs, re)
	}

	return rgs, nil
}
