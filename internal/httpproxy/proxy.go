package httpproxy

import (
	"net/http"

	"github.com/sirupsen/logrus"
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
func (proxy *Proxy) RegisterRoutes(mux *http.ServeMux, prefix string, skipLogs bool) {
	for _, s := range *proxy {
		mux.Handle(prefix+"proxywrite/"+s.Name, LogH(skipLogs, http.HandlerFunc(s.write)))
		mux.Handle(prefix+"proxyread/"+s.Name, LogH(skipLogs, http.HandlerFunc(s.read)))
	}
}

// TODO: Move to utility package?
func LogH(skipLogs bool, next http.Handler) http.Handler {
	if skipLogs {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logrus.Debugf("http-server: %s: %q", r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}
