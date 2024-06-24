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
	// TODO: Implement
	fmt.Println("=== MESSAGE INDEX ===")
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
	// TODO: Implement
	fmt.Println("=== MULTIPART INDEX ===", idx)
}

func (h *smtpHTTPHandler) handleMultipartMeta(
	w http.ResponseWriter, r *http.Request, idx, partIdx int,
) {
	// TODO: Implement
	fmt.Println("=== MULTIPART META ===", idx, partIdx)
}

func (h *smtpHTTPHandler) handleMultipartBody(
	w http.ResponseWriter, r *http.Request, idx, partIdx int,
) {
	// TODO: Implement
	fmt.Println("=== MULTIPART BODY ===", idx, partIdx)
}
