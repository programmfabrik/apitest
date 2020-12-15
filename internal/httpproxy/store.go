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

// errorResponse definition
type errorResponse struct {
	Error string `json:"error"`
}

// request definition
type request struct {
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Headers http.Header `json:"header"`
	Query   url.Values  `json:"query"`
	Body    []byte      `json:"body"`
}

// storeEntry definition
type storeEntry struct {
	Offset  int
	Request request
}

// StoreConfig definition
type StoreConfig struct {
	Mode Mode `json:"mode"`
}

// store definition
type store struct {
	Name string
	Mode Mode
	Data []storeEntry
}

// write stores incoming request data
func (st *store) write(w http.ResponseWriter, r *http.Request) {

	var err error

	reqData := request{
		Method:  r.Method,
		Path:    r.URL.Path,
		Headers: r.Header,
		Query:   r.URL.Query(),
	}

	offset := len(st.Data)

	if r.Body != nil {
		reqData.Body, err = ioutil.ReadAll(r.Body)
		if err != nil {
			respondWithErr(w, http.StatusInternalServerError, errors.Errorf("Could not read request body: %s", err))
			return
		}
	}

	st.Data = append(st.Data, storeEntry{offset, reqData})

	err = json.NewEncoder(w).Encode(struct {
		Offset int `json:"offset"`
	}{offset})
	if err != nil {
		respondWithErr(w, http.StatusInternalServerError, errors.Errorf("Could not encode response: %s", err))
	}
}

// read reads existing requests stored data
func (st *store) read(w http.ResponseWriter, r *http.Request) {

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

	count := len(st.Data)
	if offset >= count {
		respondWithErr(w, http.StatusBadRequest, errors.Errorf("Offset (%d) is higher than count (%d)", offset, count))
		return
	}

	nextOffset := offset + 1
	if nextOffset >= count {
		nextOffset = 0
	}
	data := st.Data[offset]

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
	err2 := json.NewEncoder(w).Encode(errorResponse{err.Error()})
	if err2 != nil {
		log.Printf("Could not encode the error (%s) response itself: %s", err, err2)
	}
}
