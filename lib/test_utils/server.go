package test_utils

import (
	"net/http"
	"net/http/httptest"
)

type handle func(*http.ResponseWriter, *http.Request)
type Routes map[string]handle

func handleWithRoute(w http.ResponseWriter, r *http.Request, routes Routes) {
	path := r.URL.Path
	handle, ok := routes[path]
	if !ok {
		w.WriteHeader(500)
		return
	}
	handle(&w, r)
}

func NewTestServer(routes Routes) *httptest.Server {
	routingHandle := func(w http.ResponseWriter, r *http.Request) {
		handleWithRoute(w, r, routes)
	}
	return httptest.NewServer(http.HandlerFunc(routingHandle))
}
