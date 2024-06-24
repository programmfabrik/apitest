package smtp

import (
	"fmt"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/programmfabrik/apitest/internal/httpproxy"
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

	mux.Handle(handler.prefix, httpproxy.LogH(skipLogs, handler))
	mux.Handle(handler.prefix+"/", httpproxy.LogH(skipLogs, handler))
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

	// We know that pathParts must have at least length 1, since empty path
	// was already handled above.
	idx, err := strconv.Atoi(pathParts[0])
	if err != nil {
		httpproxy.RespondWithErr(
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
		}
	case 3, 4:
		if pathParts[1] == "multipart" {
			partIdx, err := strconv.Atoi(pathParts[2])
			if err != nil {
				httpproxy.RespondWithErr(
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

func (h *smtpHTTPHandler) handleMessageIndex(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement search function

	receivedMessages := h.server.ReceivedMessages()

	messagesOut := make([]any, 0)

	for i, msg := range receivedMessages {
		messagesOut = append(messagesOut, map[string]any{
			"idx":        i,
			"receivedAt": msg.ReceivedAt(),
		})
	}

	out := make(map[string]any)
	out["count"] = len(receivedMessages)
	out["messages"] = messagesOut

	httpproxy.RespondWithJSON(w, http.StatusOK, out)
}

func (h *smtpHTTPHandler) handleMessageMeta(w http.ResponseWriter, r *http.Request, idx int) {
	// TODO: Implement
	fmt.Println("=== MESSAGE META ===", idx)
}

func (h *smtpHTTPHandler) handleMessageBody(w http.ResponseWriter, r *http.Request, idx int) {
	// TODO: Implement
	fmt.Println("=== MESSAGE BODY ===", idx)
}

func (h *smtpHTTPHandler) handleMultipartIndex(w http.ResponseWriter, r *http.Request, idx int) {
	// TODO: Implement search function
	msg := h.retrieveMessage(w, idx)
	if msg == nil {
		return
	}
	if !ensureIsMultipart(w, msg) {
		return
	}

	multiparts := msg.Multiparts()

	multipartsOut := make([]any, 0)

	for i, part := range multiparts {
		multipartsOut = append(multipartsOut, buildMultipartMeta(part, i))
	}

	out := make(map[string]any)
	out["count"] = len(multiparts)
	out["multiparts"] = multipartsOut

	httpproxy.RespondWithJSON(w, http.StatusOK, out)
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

	// TODO: Implement
	fmt.Println("=== MULTIPART META ===", idx, partIdx)
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

	// TODO: Implement
	fmt.Println("=== MULTIPART BODY ===", idx, partIdx)
}

// retrieveMessage tries to retrieve the ReceivedMessage with the given index.
// If found, returns the message. If not found, responds with Status 404
// and returns nil.
func (h *smtpHTTPHandler) retrieveMessage(w http.ResponseWriter, idx int) *ReceivedMessage {
	msg, err := h.server.ReceivedMessage(idx)
	if err != nil {
		httpproxy.RespondWithErr(w, http.StatusNotFound, err)
		return nil
	}

	return msg
}

func buildMultipartMeta(part *ReceivedPart, partIdx int) map[string]any {
	out := map[string]any{
		"idx": partIdx,
	}

	for k, v := range part.Headers() {
		out[k] = v
	}

	return out
}

// ensureIsMultipart checks whether the referenced message is a multipart
// message, returns true and does nothing further if so, returns false after
// replying with Status 404 if not.
func ensureIsMultipart(w http.ResponseWriter, msg *ReceivedMessage) bool {
	if msg.IsMultipart() {
		return true
	}

	httpproxy.RespondWithErr(w, http.StatusNotFound, fmt.Errorf(
		"multipart information was requested for non-multipart message",
	))

	return false
}
