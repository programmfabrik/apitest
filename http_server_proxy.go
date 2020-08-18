package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

// HTTPServerProxyStoreMode definition
type HTTPServerProxyStoreMode string

const (
	// HTTPServerProxyStoreModePassthrough mode
	HTTPServerProxyStoreModePassthrough HTTPServerProxyStoreMode = "passthru"
)

// HTTPServerProxyStoreErrorResponseBody definition
type HTTPServerProxyStoreErrorResponseBody struct {
	Error string `json:"error"`
}

// HTTPServerProxyStoreRequestData definition
type HTTPServerProxyStoreRequestData struct {
	Method  string      `json:"method"`
	Path    string      `json:"path"`
	Headers http.Header `json:"header"`
	Query   url.Values  `json:"query"`
	Body    []byte      `json:"body"`
}

// HTTPServerProxyStoreResponseData definition
type HTTPServerProxyStoreResponseData struct {
	StatusCode int                                  `json:"statuscode"`
	Headers    http.Header                          `json:"header"`
	Body       HTTPServerProxyStoreResponseDataBody `json:"body"`
}

// HTTPServerProxyStoreResponseDataBody definition
type HTTPServerProxyStoreResponseDataBody struct {
	Offset int `json:"offset"`
}

// HTTPServerProxyStoreDataEntry definition
type HTTPServerProxyStoreDataEntry struct {
	Offset   int                              `json:"offset"`
	Request  HTTPServerProxyStoreRequestData  `json:"request"`
	Response HTTPServerProxyStoreResponseData `json:"response"`
}

// HTTPServerProxyStore definition
type HTTPServerProxyStore struct {
	Name string                   `json:"name"`
	Mode HTTPServerProxyStoreMode `json:"mode"`
	Data []HTTPServerProxyStoreDataEntry
}

// HTTPServerProxy definition
type HTTPServerProxy map[string]HTTPServerProxyStore

// NewProxy allocates a new http proxy from its configuration
func NewProxy(cfg HTTPServerProxy) *HTTPServerProxy {
	proxy := new(HTTPServerProxy)
	*proxy = cfg
	return proxy
}

// Listen for the proxy store/retrieve
func (proxy *HTTPServerProxy) Listen(mux *http.ServeMux, prefix string) {
	for p, s := range *proxy {
		store := HTTPServerProxyStore{Name: p, Mode: s.Mode, Data: []HTTPServerProxyStoreDataEntry{}}
		mux.HandleFunc(fmt.Sprintf("%swrite/%s", prefix, p), store.write)
		mux.HandleFunc(fmt.Sprintf("%sread/%s", prefix, p), store.read)
	}
}

// write stores incoming request data
func (store *HTTPServerProxyStore) write(w http.ResponseWriter, r *http.Request) {

	reqData := HTTPServerProxyStoreRequestData{
		Method:  r.Method,
		Path:    r.URL.RequestURI(),
		Headers: r.Header,
		Query:   r.URL.Query(),
	}

	offset := len(store.Data)

	resData := HTTPServerProxyStoreResponseData{
		StatusCode: http.StatusOK,
		Headers: http.Header{
			"Content-Type": []string{"application/json"},
		},
		Body: HTTPServerProxyStoreResponseDataBody{
			Offset: offset,
		},
	}

	if r.Body != nil {
		bin, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err := json.NewEncoder(w).Encode(HTTPServerProxyStoreErrorResponseBody{
				Error: fmt.Sprintf("Could not read request body: %s", err),
			})
			if err != nil {
				log.Printf("Could not encode even the error response: %s", err)
			}
			return
		}
		reqData.Body = bin
	}

	store.Data = append(store.Data, HTTPServerProxyStoreDataEntry{offset, reqData, resData})

	err := json.NewEncoder(w).Encode(resData.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err := json.NewEncoder(w).Encode(HTTPServerProxyStoreErrorResponseBody{
			Error: fmt.Sprintf("Could not encode response: %s", err),
		})
		if err != nil {
			log.Printf("Could not encode even the error response: %s", err)
		}
	}
}

// read reads existing requests stored data
func (store *HTTPServerProxyStore) read(w http.ResponseWriter, r *http.Request) {

	var (
		err    error
		offset int
	)

	q := r.URL.Query()
	offsetStr := q.Get("offset")
	if offsetStr != "" {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			err = json.NewEncoder(w).Encode(HTTPServerProxyStoreErrorResponseBody{
				Error: fmt.Sprintf("Invalid offset %s", offsetStr),
			})
			if err != nil {
				log.Printf("Could not encode even the error response: %s", err)
			}
			return
		}
	}

	count := len(store.Data)
	if offset >= count {
		w.WriteHeader(http.StatusBadRequest)
		err = json.NewEncoder(w).Encode(HTTPServerProxyStoreErrorResponseBody{
			Error: fmt.Sprintf("Offset (%d) is higher than count (%d)", offset, count),
		})
		if err != nil {
			log.Printf("Could not encode even the error response: %s", err)
		}
		return
	}

	nextOffset := offset + 1
	if nextOffset >= count {
		nextOffset = 0
	}
	data := store.Data[offset]

	req := data.Request

	w.Header().Add("X-Proxy-Store-Count", fmt.Sprintf("%d", count))
	w.Header().Add("X-Proxy-Store-Next-Offset", fmt.Sprintf("%d", nextOffset))
	w.Header().Add("X-Request-Method", req.Method)
	w.Header().Add("X-Request-Path", req.Path)
	w.Header().Add("X-Request-Query", req.Query.Encode())
	for k, v := range req.Headers {
		for _, h := range v {
			w.Header().Add(fmt.Sprintf("X-Request-%s", k), h)
		}
	}
	w.Header().Add("Content-Type", req.Headers.Get("Content-Type"))

	_, err = w.Write(req.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		err := json.NewEncoder(w).Encode(HTTPServerProxyStoreErrorResponseBody{
			Error: fmt.Sprintf("Could not encode response: %s", err),
		})
		if err != nil {
			log.Printf("Could not encode even the error response: %s", err)
		}
	}
}
