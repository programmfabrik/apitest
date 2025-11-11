package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"unicode/utf8"

	"github.com/pkg/errors"
	"github.com/programmfabrik/apitest/internal/httpproxy"
	"github.com/programmfabrik/apitest/pkg/lib/util"
	"github.com/programmfabrik/golib"
	"github.com/sirupsen/logrus"
)

// startHttpServer start a simple http server that can server local test resources during the testsuite is running
func (ats *Suite) startHttpServer() {

	if ats.HttpServer == nil || ats.httpServer != nil {
		return
	}

	// TODO: Can we remove idleConnsClosed, because it does not seem to do anything?
	ats.idleConnsClosed = make(chan struct{})
	mux := http.NewServeMux()

	if ats.HttpServer.Dir == "" {
		ats.httpServerDir = ats.manifestDir
	} else if filepath.IsAbs(ats.HttpServer.Dir) {
		ats.httpServerDir = filepath.Clean(ats.HttpServer.Dir)
	} else {
		ats.httpServerDir = filepath.Join(ats.manifestDir, ats.HttpServer.Dir)
	}
	mux.Handle("/", logH(ats.config.logShort, customStaticHandler(http.FileServer(http.Dir(ats.httpServerDir)))))

	// bounce json response
	mux.Handle("/bounce-json", logH(ats.config.logShort, cookiesMiddleware(http.HandlerFunc(bounceJSON))))

	// bounce binary response with information in headers
	mux.Handle("/bounce", logH(ats.config.logShort, http.HandlerFunc(bounceBinary)))

	// bounce query response with query in response body, as it is
	mux.Handle("/bounce-query", logH(ats.config.logShort, http.HandlerFunc(bounceQuery)))

	// Start listening into proxy
	ats.httpServerProxy = httpproxy.New(ats.HttpServer.Proxy)
	ats.httpServerProxy.RegisterRoutes(mux, "/", ats.config.logShort)

	// Register SMTP server query routes
	if ats.smtpServer != nil {
		ats.smtpServer.RegisterRoutes(mux, "/", ats.config.logShort)
	}

	ats.httpServer = &http.Server{
		Addr:    ats.HttpServer.Addr,
		Handler: mux,
	}

	run := func() {
		if !ats.config.logShort {
			logrus.Infof("Starting HTTP Server: %s: %s", ats.HttpServer.Addr, ats.httpServerDir)
		}

		err := ats.httpServer.ListenAndServe()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			// Error starting or closing listener:
			logrus.Fatalf("HTTP server ListenAndServe: %s", err.Error())
		}
	}

	if ats.HttpServer.Testmode {
		// Run in foreground to test
		logrus.Infof("Testmode for HTTP Server. Listening, not running tests...")
		run()
	} else {
		go run()

		util.WaitForTCP(ats.HttpServer.Addr)
	}
}

// customStaticHandler can perform some operations before passing into final handler
func customStaticHandler(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		qs := r.URL.Query()
		// We try not to include Content-Length header here
		// As ultimately the default FileServer implementation will override all of them
		// After diving into its code, the only way to avoid it is setting Content-Encoding header to some value
		// In this case, 'identity', as per RFC 7231 / RFC 2616, means no compression or modification
		noContentLengthHeader := qs.Get("no-content-length")
		if noContentLengthHeader == "1" || noContentLengthHeader == "true" {
			w.Header().Set("Content-Encoding", "identity")
		}
		h.ServeHTTP(w, r)
	}
}

// stopHttpServer stop the http server that was started for this test suite
func (ats *Suite) stopHttpServer() {

	if ats.HttpServer == nil || ats.httpServer == nil {
		return
	}

	err := ats.httpServer.Shutdown(context.Background())
	if err != nil {
		// Error from closing listeners, or context timeout:
		logrus.Errorf("HTTP server Shutdown: %v", err)
		close(ats.idleConnsClosed)
		<-ats.idleConnsClosed
	} else if !ats.config.logShort {
		logrus.Infof("Http Server stopped: %s", ats.httpServerDir)
	}

	ats.httpServer = nil
}

type errorResponse struct {
	Error string `json:"error"`
	Body  any    `json:"body,omitempty"`
}

func formatErrorResponse(w http.ResponseWriter, statuscode int, err error, body any) {
	resp := errorResponse{
		Error: err.Error(),
		Body:  body,
	}

	b, err2 := golib.JsonBytesIndent(resp, "", "  ")
	if err2 != nil {
		logrus.Debugf("Could not marshall error message %s: %s", err.Error(), err2.Error())
		http.Error(w, err2.Error(), 500)
	}

	http.Error(w, string(b), statuscode)
}

type bounceResponse struct {
	Header      http.Header `json:"header"`
	QueryParams url.Values  `json:"query_params"`
	Body        any         `json:"body"`
}

// bounceJSON builds a json response including the header, query params and body of the request
func bounceJSON(w http.ResponseWriter, r *http.Request) {

	var (
		err                 error
		bodyBytes           []byte
		bodyJSON, errorBody any
	)

	bodyBytes, err = io.ReadAll(r.Body)

	if utf8.Valid(bodyBytes) {
		if len(bodyBytes) > 0 {
			errorBody = string(bodyBytes)
		} else {
			errorBody = nil
		}
	} else {
		errorBody = bodyBytes
	}

	if err != nil {
		formatErrorResponse(w, 500, err, errorBody)
		return
	}

	response := bounceResponse{
		Header:      r.Header,
		QueryParams: r.URL.Query(),
	}
	if len(bodyBytes) > 0 {
		err = json.Unmarshal(bodyBytes, &bodyJSON)
		if err != nil {
			formatErrorResponse(w, 500, err, errorBody)
			return
		}
		response.Body = bodyJSON
	}

	responseData, err := golib.JsonBytesIndent(response, "", "  ")
	if err != nil {
		formatErrorResponse(w, 500, err, response)
		return
	}

	w.Write(responseData)
}

// bounceBinary returns the request in binary form
func bounceBinary(w http.ResponseWriter, r *http.Request) {

	for param, values := range r.URL.Query() {
		for _, value := range values {
			w.Header().Add("X-Req-Query-"+param, value)
		}
	}

	for param, values := range r.Header {
		for _, value := range values {
			w.Header().Add("X-Req-Header-"+param, value)
		}
	}

	io.Copy(w, r.Body)
}

// bounceQuery returns the request query in response body
// for those cases where a body cannt be provided
func bounceQuery(w http.ResponseWriter, r *http.Request) {
	rBody := bytes.NewBufferString(r.URL.RawQuery)
	io.Copy(w, rBody)
}

func cookiesMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ckHeader := r.Header.Values("X-Test-Set-Cookies")
		for _, ck := range ckHeader {
			w.Header().Add("Set-Cookie", ck)
		}
		next.ServeHTTP(w, r)
	})
}

func logH(skipLog bool, next http.Handler) http.Handler {
	if skipLog {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// logrus.Debugf("http-server: %s: %q", r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}
