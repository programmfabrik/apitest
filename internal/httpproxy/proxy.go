package httpproxy

import (
	"fmt"
	"net/http"
)

// ProxyConfig definition
type ProxyConfig map[string]StoreConfig

// Proxy definition
type Proxy map[string]*Store

// NewProxy allocates a new http proxy from its configuration
func NewProxy(cfg ProxyConfig) *Proxy {
	proxy := Proxy{}
	for k, v := range cfg {
		proxy[k] = &Store{k, v.Mode, []storeEntry{}}
	}
	return &proxy
}

// Listen for the proxy store/retrieve
func (proxy *Proxy) Listen(mux *http.ServeMux, prefix string) {
	for _, s := range *proxy {
		mux.HandleFunc(fmt.Sprintf("%sproxywrite/%s", prefix, s.Name), s.write)
		mux.HandleFunc(fmt.Sprintf("%sproxyread/%s", prefix, s.Name), s.read)
	}
}
