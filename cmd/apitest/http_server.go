package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/sirupsen/logrus"
	"golang.org/x/xerrors"
)

// StartHttpServer start a simple http server that can server local test resources during the testsuite is running
func (ats *Suite) StartHttpServer() {

	if ats.HttpServer == nil {
		return
	}

	ats.idleConnsClosed = make(chan struct{})
	mux := http.NewServeMux()

	if ats.HttpServer.Dir == "" {
		ats.httpServerDir = ats.manifestDir
	} else {
		ats.httpServerDir = filepath.Clean(ats.manifestDir + "/" + ats.HttpServer.Dir)
	}
	mux.Handle("/", http.FileServer(http.Dir(ats.httpServerDir)))

	// read the file at query param 'path' and return it as the response body
	mux.HandleFunc("/load-file", func(w http.ResponseWriter, r *http.Request) {
		loadFile(w, r, ats.httpServerDir)
	})

	// bounce the request body back as the response body
	mux.HandleFunc("/bounce", bounce)

	// bounce the request url parameters back as the response body
	mux.HandleFunc("/bounce-query", bounceQuery)

	// bounce the request headers back as the response body
	mux.HandleFunc("/bounce-header", bounceHeader)

	ats.httpServer = http.Server{
		Addr:    ats.HttpServer.Addr,
		Handler: mux,
	}

	run := func() {
		logrus.Infof("Starting HTTP Server: %s: %s", ats.HttpServer.Addr, ats.httpServerDir)

		err := ats.httpServer.ListenAndServe()
		if err != http.ErrServerClosed {
			// Error starting or closing listener:
			logrus.Errorf("HTTP server ListenAndServe: %v", err)
			return
		}
	}

	if ats.HttpServer.Testmode {
		// Run in foreground to test
		logrus.Infof("Testmode for HTTP Server. Listening, not running tests...")
		run()
	} else {
		go run()
	}
}

// StopHttpServer stop the http server that was started for this test suite
func (ats *Suite) StopHttpServer() {

	if ats.HttpServer == nil {
		return
	}

	err := ats.httpServer.Shutdown(context.Background())
	if err != nil {
		// Error from closing listeners, or context timeout:
		logrus.Errorf("HTTP server Shutdown: %v", err)
		close(ats.idleConnsClosed)
		<-ats.idleConnsClosed
	} else {
		logrus.Infof("Http Server stopped: %s", ats.httpServerDir)
	}
	return
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func errorResponse(w http.ResponseWriter, statuscode int, err error) {
	resp := ErrorResponse{
		Error: err.Error(),
	}

	b, err2 := json.MarshalIndent(resp, "", "  ")
	if err2 != nil {
		logrus.Debugf("Could not marshall error message %s: %s", err, err2)
		http.Error(w, err2.Error(), 500)
	}

	http.Error(w, string(b), statuscode)
}

// loadFile reads the file at query param 'path' and returns it as the response body
func loadFile(w http.ResponseWriter, r *http.Request, dir string) {
	fn := r.URL.Query().Get("path")
	if fn == "" {
		errorResponse(w, 400, xerrors.Errorf("path not found in query_params"))
		return
	}

	fpath := dir + "/" + fn

	of, err := os.Open(fpath)
	defer of.Close()
	if err != nil {
		errorResponse(w, 404, xerrors.Errorf("file %s not found", fpath))
		return
	}

	// build headers
	w.Header().Set("Content-Disposition", "attachment; filename="+fpath)

	// Content-Type
	fh := make([]byte, 512)
	of.Read(fh)
	contentType := http.DetectContentType(fh)
	w.Header().Set("Content-Type", contentType)

	// Content-Length
	fs, _ := of.Stat()
	w.Header().Set("Content-Length", strconv.FormatInt(fs.Size(), 10))

	// write body
	of.Seek(0, 0)
	io.Copy(w, of)
}

// bounce reads the body from the requests and writes it directly to the response body
func bounce(w http.ResponseWriter, r *http.Request) {
	io.Copy(w, r.Body)
}

// bounceQuery builds a json response body from the url parameters
func bounceQuery(w http.ResponseWriter, r *http.Request) {
	params := map[string]string{}
	for k, v := range r.URL.Query() {
		params[k] = v[0]
	}

	body, err := json.MarshalIndent(params, "", "  ")
	if err != nil {
		errorResponse(w, 500, err)
	}

	w.Write(body)
}

// bounceHeader builds a json response body the request header
func bounceHeader(w http.ResponseWriter, r *http.Request) {
	header := map[string]string{}
	for k, v := range r.Header {
		header[k] = v[0]
	}

	body, err := json.MarshalIndent(header, "", "  ")
	if err != nil {
		errorResponse(w, 500, err)
	}

	w.Write(body)
}
