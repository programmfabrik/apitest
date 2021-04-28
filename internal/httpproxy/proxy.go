package httpproxy

import (
	"fmt"
	"net/http"
)

// ProxyConfig definition
type ProxyConfig map[string]storeConfig

// Proxy definition
type Proxy map[string]*store

// NewProxy allocates a new http proxy from its configuration
func New(cfg ProxyConfig) *Proxy {
	proxy := Proxy{}
	for k, v := range cfg {
		proxy[k] = &store{k, v.Mode, []storeEntry{}}
	}
	return &proxy
}

// RegisterRoutes for the proxy store/retrieve
func (proxy *Proxy) RegisterRoutes(mux *http.ServeMux, prefix string) {
	for _, s := range *proxy {
		mux.HandleFunc(fmt.Sprintf("%sproxywrite/%s", prefix, s.Name), s.write)
		mux.HandleFunc(fmt.Sprintf("%sproxyread/%s", prefix, s.Name), s.read)
	}
}
