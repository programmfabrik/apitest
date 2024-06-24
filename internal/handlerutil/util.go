package handlerutil

import (
	"encoding/json"
	"net/http"

	"github.com/sirupsen/logrus"
)

// errorResponse definition
type errorResponse struct {
	Error string `json:"error"`
}

// LogH is a middleware that logs requests via logrus.Debugf.
func LogH(skipLogs bool, next http.Handler) http.Handler {
	if skipLogs {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logrus.Debugf("http-server: %s: %q", r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

// RespondWithErr responds using a JSON-serialized error message.
func RespondWithErr(w http.ResponseWriter, status int, err error) {
	RespondWithJSON(w, status, errorResponse{err.Error()})
}

// RespondWithJSON responds with a JSON-serialized value.
func RespondWithJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)

	err := json.NewEncoder(w).Encode(v)
	if err != nil {
		logrus.Errorf("Could not encode JSON response: %s (%v)", err, v)
	}
}
