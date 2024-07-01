package smtp

import (
	_ "embed"
	"fmt"
	"html/template"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"

	"github.com/programmfabrik/apitest/internal/handlerutil"
	"github.com/sirupsen/logrus"
)

//go:embed gui.html
var guiTemplateSrc string
var guiTemplate = template.Must(template.New("gui").Parse(guiTemplateSrc))

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

	if pathParts[0] == "gui" {
		h.handleGUI(w, r)
		return
	}

	idx, err := strconv.Atoi(pathParts[0])
	if err != nil {
		handlerutil.RespondWithErr(
			w, http.StatusBadRequest,
			fmt.Errorf("could not parse message index: %w", err),
		)
		return
	}

	switch len(pathParts) {
	case 1:
		h.handleMessageMeta(w, r, idx)
		return
	case 2:
		switch pathParts[1] {
		case "body":
			h.handleMessageBody(w, r, idx)
			return
		case "multipart":
			h.handleMultipartIndex(w, r, idx)
			return
		case "raw":
			h.handleRawMessageData(w, r, idx)
			return
		}
	case 3, 4:
		if pathParts[1] == "multipart" {
			partIdx, err := strconv.Atoi(pathParts[2])
			if err != nil {
				handlerutil.RespondWithErr(
					w, http.StatusBadRequest,
					fmt.Errorf("could not parse multipart index: %w", err),
				)
				return
			}

			if len(pathParts) == 3 {
				h.handleMultipartMeta(w, r, idx, partIdx)
				return
			} else if pathParts[3] == "body" {
				h.handleMultipartBody(w, r, idx, partIdx)
				return
			}
		}
	}

	// If routing failed, return status 404.
	w.WriteHeader(http.StatusNotFound)
}

func (h *smtpHTTPHandler) handleGUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	err := guiTemplate.Execute(w, map[string]any{"prefix": h.prefix})
	if err != nil {
		logrus.Error("error rendering GUI:", err)
	}
}

func (h *smtpHTTPHandler) handleMessageIndex(w http.ResponseWriter, r *http.Request) {
	var receivedMessages []*ReceivedMessage

	headerSearchRgx, err := extractSearchRegex(w, r.URL.Query(), "header")
	if err != nil {
		handlerutil.RespondWithErr(w, http.StatusBadRequest, err)
		return
	}
	if headerSearchRgx == nil {
		receivedMessages = h.server.ReceivedMessages()
	} else {
		receivedMessages = h.server.SearchByHeader(headerSearchRgx)
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

func (h *smtpHTTPHandler) handleMessageMeta(w http.ResponseWriter, r *http.Request, idx int) {
	msg := h.retrieveMessage(w, idx)
	if msg == nil {
		return
	}

	out := buildMessageBasicMeta(msg)

	out["body_size"] = len(msg.Body())

	headers := make(map[string]any)
	for k, v := range msg.Headers() {
		headers[k] = v
	}
	out["headers"] = headers

	handlerutil.RespondWithJSON(w, http.StatusOK, out)
}

func (h *smtpHTTPHandler) handleMessageBody(w http.ResponseWriter, r *http.Request, idx int) {
	msg := h.retrieveMessage(w, idx)
	if msg == nil {
		return
	}

	contentType, ok := msg.Headers()["Content-Type"]
	if ok {
		w.Header()["Content-Type"] = contentType
	}

	w.Write(msg.Body())
}

func (h *smtpHTTPHandler) handleMultipartIndex(w http.ResponseWriter, r *http.Request, idx int) {
	msg := h.retrieveMessage(w, idx)
	if msg == nil {
		return
	}
	if !ensureIsMultipart(w, msg) {
		return
	}

	var multiparts []*ReceivedPart
	headerSearchRgx, err := extractSearchRegex(w, r.URL.Query(), "header")
	if err != nil {
		handlerutil.RespondWithErr(w, http.StatusBadRequest, err)
		return
	}
	if headerSearchRgx == nil {
		multiparts = msg.Multiparts()
	} else {
		multiparts = msg.SearchPartsByHeader(headerSearchRgx)
	}

	multipartsOut := make([]any, 0)

	for _, part := range multiparts {
		multipartsOut = append(multipartsOut, buildMultipartMeta(part))
	}

	out := make(map[string]any)
	out["count"] = len(multiparts)
	out["multiparts"] = multipartsOut

	handlerutil.RespondWithJSON(w, http.StatusOK, out)
}

func (h *smtpHTTPHandler) handleMultipartMeta(
	w http.ResponseWriter, r *http.Request, idx, partIdx int,
) {
	msg := h.retrieveMessage(w, idx)
	if msg == nil {
		return
	}
	if !ensureIsMultipart(w, msg) {
		return
	}
	part := retrievePart(w, msg, partIdx)
	if part == nil {
		return
	}

	handlerutil.RespondWithJSON(w, http.StatusOK, buildMultipartMeta(part))
}

func (h *smtpHTTPHandler) handleMultipartBody(
	w http.ResponseWriter, r *http.Request, idx, partIdx int,
) {
	msg := h.retrieveMessage(w, idx)
	if msg == nil {
		return
	}
	if !ensureIsMultipart(w, msg) {
		return
	}
	part := retrievePart(w, msg, partIdx)
	if part == nil {
		return
	}

	contentType, ok := part.Headers()["Content-Type"]
	if ok {
		w.Header()["Content-Type"] = contentType
	}

	w.Write(part.Body())
}

func (h *smtpHTTPHandler) handleRawMessageData(w http.ResponseWriter, r *http.Request, idx int) {
	msg := h.retrieveMessage(w, idx)
	if msg == nil {
		return
	}

	w.Header().Set("Content-Type", "message/rfc822")

	w.Write(msg.RawMessageData())
}

// retrieveMessage tries to retrieve the ReceivedMessage with the given index.
// If found, returns the message. If not found, responds with Status 404
// and returns nil.
func (h *smtpHTTPHandler) retrieveMessage(w http.ResponseWriter, idx int) *ReceivedMessage {
	msg, err := h.server.ReceivedMessage(idx)
	if err != nil {
		handlerutil.RespondWithErr(w, http.StatusNotFound, err)
		return nil
	}

	return msg
}

// retrievePart tries to retrieve the ReceivedPart with the given index.
// If found, returns the part. If not found, responds with Status 404
// and returns nil.
func retrievePart(w http.ResponseWriter, msg *ReceivedMessage, partIdx int) *ReceivedPart {
	multiparts := msg.Multiparts()

	if partIdx >= len(multiparts) {
		handlerutil.RespondWithErr(w, http.StatusNotFound, fmt.Errorf(
			"ReceivedMessage does not contain multipart with index %d", partIdx,
		))
		return nil
	}

	return msg.Multiparts()[partIdx]
}

func buildMessageBasicMeta(msg *ReceivedMessage) map[string]any {
	out := map[string]any{
		"idx":         msg.Index(),
		"isMultipart": msg.IsMultipart(),
		"receivedAt":  msg.ReceivedAt(),
	}

	from, ok := msg.Headers()["From"]
	if ok {
		out["from"] = from
	}

	to, ok := msg.Headers()["To"]
	if ok {
		out["to"] = to
	}

	subject, ok := msg.Headers()["Subject"]
	if ok && len(subject) == 1 {
		out["subject"] = subject[0]
	}

	return out
}

func buildMultipartMeta(part *ReceivedPart) map[string]any {
	out := map[string]any{
		"idx":       part.Index(),
		"body_size": len(part.Body()),
	}

	headers := make(map[string]any)
	for k, v := range part.Headers() {
		headers[k] = v
	}
	out["headers"] = headers

	return out
}

// ensureIsMultipart checks whether the referenced message is a multipart
// message, returns true and does nothing further if so, returns false after
// replying with Status 404 if not.
func ensureIsMultipart(w http.ResponseWriter, msg *ReceivedMessage) bool {
	if msg.IsMultipart() {
		return true
	}

	handlerutil.RespondWithErr(w, http.StatusNotFound, fmt.Errorf(
		"multipart information was requested for non-multipart message",
	))

	return false
}

// extractSearchRegex tries to extract a regular expression from the referenced
// query parameter. If no query parameter is given and otherwise no error has
// occurred, this function returns (nil, nil).
func extractSearchRegex(
	w http.ResponseWriter, queryParams map[string][]string, paramName string,
) (*regexp.Regexp, error) {
	searchParam, ok := queryParams[paramName]
	if ok {
		if len(searchParam) != 1 {
			return nil, fmt.Errorf(
				"Encountered multiple %q params", paramName,
			)
		}

		re, err := regexp.Compile(searchParam[0])
		if err != nil {
			return nil, fmt.Errorf(
				"could not compile %q regex: %w", paramName, err,
			)
		}

		return re, nil
	}

	return nil, nil
}
