package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
)

// HTTPServerProxyMode type
type HTTPServerProxyMode string
const (
	// HTTPServerProxyModePassthrough mode
	HTTPServerProxyModePassthrough HTTPServerProxyMode = "passthru"
)

// HTTPServerProxy 
type HTTPServerProxy map[string]HTTPServerProxyOptions

// HTTPServerProxyOptions struct
type HTTPServerProxyOptions struct {
	Mode HTTPServerProxyMode `json:"mode"`
}

// NewHTTPServerProxy creates a new proxy and returns its address, or an error
func NewHTTPServerProxy(readPrefixPath string, writePrefixPath string, basePath string, mode HTTPServerProxyMode) (*HTTPServerProxy, error) {
	if basePath == "" {
		return nil, errors.New("Base Path cannot be empty")
	}
	if readPrefixPath == writePrefixPath {
		return nil, errors.Errorf("Read prefix (%s) cannot be the same as write (%s)", readPrefixPath, writePrefixPath)
	}
	switch mode {
	case HTTPServerProxyModePassthrough:
	default:
		return nil, errors.Errorf("Proxy mode %s not supported", mode)
	}
	return &HTTPServerProxy{readPrefixPath, writePrefixPath, basePath, mode}, nil
}

// Listen sets up routes and listeners for the proxy
func (proxy *HTTPServerProxy) Listen(router *mux.Router) {

	fullWritePath := fmt.Sprintf("%s/%s", proxy.WritePrefixPath, proxy.BasePath)
	router.HandleFunc(fullWritePath, proxy.write)

	fullReadPath := fmt.Sprintf("%s/%s", proxy.ReadPrefixPath, proxy.BasePath)
	router.HandleFunc(fullReadPath, proxy.read)
}

// write stores incoming request data
func (proxy *HTTPServerProxy) write(w http.ResponseWriter, r *http.Request) {
	// TODO:
}

// read reads existing requests stored data
func (proxy *HTTPServerProxy) read(w http.ResponseWriter, r *http.Request) {
	// TODO:
}
