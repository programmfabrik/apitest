package httpproxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
)

// Mode definition
type Mode string

const (
	// ModePassthrough mode
	ModePassthrough Mode = "passthru"
)

// ErrorResponse definition
type ErrorResponse struct {
	Error string `json:"error"`
}

// Request definition
type Request struct {
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Headers http.Header `json:"header"`
	Query   url.Values  `json:"query"`
	Body    []byte      `json:"body"`
}

// WriteResponse definition
type WriteResponse struct {
	StatusCode int               `json:"statuscode"`
	Headers    http.Header       `json:"header"`
	Body       WriteResponseBody `json:"body"`
}

// WriteResponseBody definition
type WriteResponseBody struct {
	Offset int `json:"offset"`
}

// storeEntry definition
type storeEntry struct {
	Offset  int
	Request Request
}

// StoreConfig definition
type StoreConfig struct {
	Mode Mode `json:"mode"`
}

// Store definition
type Store struct {
	Name string
	Mode Mode
	Data []storeEntry
}

// write stores incoming request data
func (store *Store) write(w http.ResponseWriter, r *http.Request) {

	var err error

	reqData := Request{
		Method:  r.Method,
		Path:    r.URL.RequestURI(),
		Headers: r.Header,
		Query:   r.URL.Query(),
	}

	offset := len(store.Data)

	resData := WriteResponse{
		StatusCode: http.StatusOK,
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: WriteResponseBody{
			Offset: offset,
		},
	}

	if r.Body != nil {
		reqData.Body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			respondWithErr(w, http.StatusInternalServerError, errors.Errorf("Could not read request body: %s", err))
			return
		}
	}

	store.Data = append(store.Data, storeEntry{offset, reqData})

	err = json.NewEncoder(w).Encode(resData.Body)
	if err != nil {
		respondWithErr(w, http.StatusInternalServerError, errors.Errorf("Could not encode response: %s", err))
	}
}

// read reads existing requests stored data
func (store *Store) read(w http.ResponseWriter, r *http.Request) {

	var (
		err    error
		offset int
	)

	q := r.URL.Query()
	offsetStr := q.Get("offset")
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			respondWithErr(w, http.StatusBadRequest, errors.Errorf("Invalid offset %s", offsetStr))
			return
		}
	}

	count := len(store.Data)
	if offset >= count {
		respondWithErr(w, http.StatusBadRequest, errors.Errorf("Offset (%d) is higher than count (%d)", offset, count))
		return
	}

	nextOffset := offset + 1
	if nextOffset >= count {
		nextOffset = 0
	}
	data := store.Data[offset]

	req := data.Request

	w.Header().Add("X-Apitest-Proxy-Store-Count", fmt.Sprintf("%d", count))
	w.Header().Add("X-Apitest-Proxy-Store-Next-Offset", fmt.Sprintf("%d", nextOffset))
	w.Header().Add("X-Apitest-Proxy-Request-Method", req.Method)
	w.Header().Add("X-Apitest-Proxy-Request-Path", req.Path)
	w.Header().Add("X-Apitest-Proxy-Request-Query", req.Query.Encode())
	for k, v := range req.Headers {
		for _, h := range v {
			w.Header().Add(k, h)
		}
	}

	_, err = w.Write(req.Body)
	if err != nil {
		respondWithErr(w, http.StatusInternalServerError, errors.Errorf("Could not encode response: %s", err))
	}
}

// respondWithErr helper
func respondWithErr(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	err2 := json.NewEncoder(w).Encode(ErrorResponse{err.Error()})
	if err2 != nil {
		log.Printf("Could not encode the error (%s) response itself: %s", err, err2)
	}
}
