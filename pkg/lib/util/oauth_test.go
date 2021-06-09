package util

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetToken(t *testing.T) {
	theToken := "thetoken"
	theClient := "my_client"
	theSecret := "foobar"
	basicAuth := theClient + ":" + theSecret
	basicAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(basicAuth))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != basicAuthHeader {
			w.WriteHeader(403)
			return
		}
		fmt.Fprintf(w, `access_token=%s`, theToken)
	}))
	defer ts.Close()

	cfg := OAuthClientConfig{
		Key: "nobody",
		Endpoint: OAuthEndpointConfig{
			TokenURL: ts.URL,
		},
		Secret: "whatever",
	}
	_, err := cfg.GetAuthToken("hey", "yo")
	if err == nil {
		t.Fatal("Expected error")
	}

	cfg = OAuthClientConfig{
		Key: theClient,
		Endpoint: OAuthEndpointConfig{
			TokenURL: ts.URL,
		},
		Secret: "whatever",
	}
	_, err = cfg.GetAuthToken("hey", "yo")
	if err == nil {
		t.Fatal("Expected error")
	}

	cfg = OAuthClientConfig{
		Key: theClient,
		Endpoint: OAuthEndpointConfig{
			TokenURL: ts.URL,
		},
		Secret: theSecret,
	}
	token, err := cfg.GetAuthToken("hey", "yo")
	if err != nil {
		t.Fatal(err)
	}
	if token == nil {
		t.Fatal("No token nor error returned")
	}
	if token.AccessToken != theToken {
		t.Fatalf("Received token: %s , expected: %s", token.AccessToken, theToken)
	}
}
