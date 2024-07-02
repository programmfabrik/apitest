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

	if pathParts[0] == "gui" && len(pathParts) == 1 {
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

	msg, err := h.server.ReceivedMessage(idx)
	if err != nil {
		handlerutil.RespondWithErr(w, http.StatusNotFound, err)
		return
	}

	if len(pathParts) == 1 {
		h.handleMessageMeta(w, r, msg)
		return
	}
	if len(pathParts) == 2 && pathParts[1] == "raw" {
		h.handleMessageRaw(w, r, msg)
		return
	}

	h.routeContentEndpoint(w, r, msg.Content(), pathParts[1:])
}

// routeContentEndpoint recursively finds a route for the remaining path parts
// based on the given ReceivedContent.
func (h *smtpHTTPHandler) routeContentEndpoint(
	w http.ResponseWriter, r *http.Request, c *ReceivedContent, remainingPathParts []string,
) {
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
			h.handleContentBody(w, r, c)
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
			h.handleMultipartMeta(w, r, part)
			return
		}

		h.routeContentEndpoint(w, r, part.Content(), remainingPathParts[2:])
		return
	}

	// If routing failed, return status 404.
	w.WriteHeader(http.StatusNotFound)
}

func (h *smtpHTTPHandler) handleContentBody(w http.ResponseWriter, r *http.Request, c *ReceivedContent) {
	contentType, ok := c.Headers()["Content-Type"]
	if ok {
		w.Header()["Content-Type"] = contentType
	}

	w.Write(c.Body())
}

func (h *smtpHTTPHandler) handleGUI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	err := guiTemplate.Execute(w, map[string]any{"prefix": h.prefix})
	if err != nil {
		logrus.Error("error rendering GUI:", err)
	}
}

func (h *smtpHTTPHandler) handleMessageIndex(w http.ResponseWriter, r *http.Request) {
	headerSearchRgx, err := extractSearchRegex(w, r.URL.Query(), "header")
	if err != nil {
		handlerutil.RespondWithErr(w, http.StatusBadRequest, err)
		return
	}

	receivedMessages := h.server.ReceivedMessages()
	if headerSearchRgx != nil {
		receivedMessages = SearchByHeader(receivedMessages, headerSearchRgx)
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

func (h *smtpHTTPHandler) handleMessageMeta(w http.ResponseWriter, r *http.Request, msg *ReceivedMessage) {
	out := buildMessageBasicMeta(msg)
	contentMeta := buildContentMeta(msg.Content())

	for k, v := range contentMeta {
		out[k] = v
	}

	handlerutil.RespondWithJSON(w, http.StatusOK, out)
}

func (h *smtpHTTPHandler) handleMessageRaw(w http.ResponseWriter, r *http.Request, msg *ReceivedMessage) {
	w.Header().Set("Content-Type", "message/rfc822")
	w.Write(msg.RawMessageData())
}

func (h *smtpHTTPHandler) handleMultipartIndex(w http.ResponseWriter, r *http.Request, c *ReceivedContent) {
	headerSearchRgx, err := extractSearchRegex(w, r.URL.Query(), "header")
	if err != nil {
		handlerutil.RespondWithErr(w, http.StatusBadRequest, err)
		return
	}

	multiparts := c.Multiparts()
	if headerSearchRgx != nil {
		multiparts = SearchByHeader(multiparts, headerSearchRgx)
	}

	handlerutil.RespondWithJSON(w, http.StatusOK, buildMultipartIndex(multiparts))
}

func (h *smtpHTTPHandler) handleMultipartMeta(
	w http.ResponseWriter, r *http.Request, part *ReceivedPart,
) {
	handlerutil.RespondWithJSON(w, http.StatusOK, buildMultipartMeta(part))
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
