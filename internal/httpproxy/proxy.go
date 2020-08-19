package httpproxy

import (
	"fmt"
	"net/http"
)

// Proxy definition
type Proxy map[string]Store

// NewProxy allocates a new http proxy from its configuration
func NewProxy(cfg Proxy) *Proxy {
	proxy := new(Proxy)
	*proxy = cfg
	return proxy
}

// Listen for the proxy store/retrieve
func (proxy *Proxy) Listen(mux *http.ServeMux, prefix string) {
	for p, s := range *proxy {
		store := Store{Name: p, Mode: s.Mode, Data: []Entry{}}
		mux.HandleFunc(fmt.Sprintf("%sproxywrite/%s", prefix, p), store.write)
		mux.HandleFunc(fmt.Sprintf("%sproxyread/%s", prefix, p), store.read)
	}
}
