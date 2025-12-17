package util

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	go_test_utils "github.com/programmfabrik/go-test-utils"
)

func TestGetPasswordCredentialsToken(t *testing.T) {
	theToken := "thetoken"
	theRefreshToken := "therefreshtoken"
	theClient := "my_client"
	theSecret := "foobar"
	basicAuth := theClient + ":" + theSecret
	basicAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(basicAuth))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != basicAuthHeader {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		fmt.Fprintf(w, `access_token=%s&refresh_token=%s`, theToken, theRefreshToken)
	}))
	defer ts.Close()

	cfg := OAuthClientConfig{
		Client: "nobody",
		Endpoint: oAuthEndpointConfig{
			TokenURL: ts.URL,
		},
		Secret: "whatever",
	}
	_, err := cfg.GetPasswordCredentialsAuthToken("hey", "yo")
	go_test_utils.ExpectError(t, err, "Expected error")

	cfg = OAuthClientConfig{
		Client: theClient,
		Endpoint: oAuthEndpointConfig{
			TokenURL: ts.URL,
		},
		Secret: "whatever",
	}
	_, err = cfg.GetPasswordCredentialsAuthToken("hey", "yo")
	go_test_utils.ExpectError(t, err, "Expected error")

	cfg = OAuthClientConfig{
		Client: theClient,
		Endpoint: oAuthEndpointConfig{
			TokenURL: ts.URL,
		},
		Secret: theSecret,
	}
	token, err := cfg.GetPasswordCredentialsAuthToken("hey", "yo")
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	if token == nil {
		t.Fatal("No token nor error returned")
	}
	if token.AccessToken != theToken {
		t.Fatalf("Received token: %s, expected: %s", token.AccessToken, theToken)
	}
	if token.RefreshToken != theRefreshToken {
		t.Fatalf("Received token: %s, expected: %s", token.AccessToken, theToken)
	}
}

func TestGetClientCredentialsToken(t *testing.T) {
	theToken := "thetoken"
	theRefreshToken := "therefreshtoken"
	theClient := "my_client"
	theSecret := "foobar"
	basicAuth := theClient + ":" + theSecret
	basicAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(basicAuth))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != basicAuthHeader {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		fmt.Fprintf(w, `access_token=%s&refresh_token=%s`, theToken, theRefreshToken)
	}))
	defer ts.Close()

	cfg := OAuthClientConfig{
		Client: "nobody",
		Endpoint: oAuthEndpointConfig{
			TokenURL: ts.URL,
		},
		Secret: "whatever",
	}
	_, err := cfg.GetClientCredentialsAuthToken()
	go_test_utils.ExpectError(t, err, "Expected error")

	cfg = OAuthClientConfig{
		Client: theClient,
		Endpoint: oAuthEndpointConfig{
			TokenURL: ts.URL,
		},
		Secret: "whatever",
	}
	_, err = cfg.GetClientCredentialsAuthToken()
	go_test_utils.ExpectError(t, err, "Expected error")

	cfg = OAuthClientConfig{
		Client: theClient,
		Endpoint: oAuthEndpointConfig{
			TokenURL: ts.URL,
		},
		Secret: theSecret,
	}
	token, err := cfg.GetClientCredentialsAuthToken()
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	if token == nil {
		t.Fatal("No token nor error returned")
	}
	if token.AccessToken != theToken {
		t.Fatalf("Received token: %s, expected: %s", token.AccessToken, theToken)
	}
	if token.RefreshToken != theRefreshToken {
		t.Fatalf("Received token: %s, expected: %s", token.AccessToken, theToken)
	}
}

func TestGetCodeToken(t *testing.T) {
	theCode := "thecode"
	theToken := "thetoken"
	theRefreshToken := "therefreshtoken"
	theUsername := "theuser"
	thePassword := "thepassword"
	theClient := "my_client"
	theSecret := "foobar"
	basicAuth := theClient + ":" + theSecret
	basicAuthHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(basicAuth))
	rs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer rs.Close()
	as := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		qv := r.URL.Query()
		guestAccess := qv.Get("guest_access")
		username := qv.Get("username")
		password := qv.Get("password")
		if guestAccess == "" && (username != theUsername || password != thePassword) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		rdURL := qv.Get("redirect_uri")
		rURL := rdURL + "?code=" + theCode
		http.Redirect(w, r, rURL, http.StatusPermanentRedirect)
	}))
	defer as.Close()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != basicAuthHeader {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.FormValue("code") != theCode {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		fmt.Fprintf(w, `access_token=%s&refresh_token=%s`, theToken, theRefreshToken)
	}))
	defer ts.Close()

	cfg := OAuthClientConfig{
		Client: "nobody",
		Endpoint: oAuthEndpointConfig{
			AuthURL:  as.URL,
			TokenURL: ts.URL,
		},
		RedirectURL: rs.URL,
		Secret:      "whatever",
	}
	_, err := cfg.GetCodeAuthToken()
	go_test_utils.ExpectError(t, err, "Expected error")

	cfg = OAuthClientConfig{
		Client: theClient,
		Endpoint: oAuthEndpointConfig{
			AuthURL:  as.URL,
			TokenURL: ts.URL,
		},
		RedirectURL: rs.URL,
		Secret:      "whatever",
	}
	_, err = cfg.GetCodeAuthToken()
	go_test_utils.ExpectError(t, err, "Expected error")

	cfg = OAuthClientConfig{
		Client: theClient,
		Endpoint: oAuthEndpointConfig{
			TokenURL: ts.URL,
		},
		Secret: theSecret,
	}
	_, err = cfg.GetCodeAuthToken("username", theUsername, "password", thePassword)
	go_test_utils.ExpectError(t, err, "Expected error")

	cfg = OAuthClientConfig{
		Client: theClient,
		Endpoint: oAuthEndpointConfig{
			AuthURL:  as.URL,
			TokenURL: ts.URL,
		},
		RedirectURL: rs.URL,
		Secret:      theSecret,
	}
	_, err = cfg.GetCodeAuthToken("username", theUsername, "password", "wrong")
	go_test_utils.ExpectError(t, err, "Expected error")

	cfg = OAuthClientConfig{
		Client: theClient,
		Endpoint: oAuthEndpointConfig{
			AuthURL:  as.URL,
			TokenURL: ts.URL,
		},
		RedirectURL: rs.URL,
		Secret:      theSecret,
	}
	token, err := cfg.GetCodeAuthToken("username", theUsername, "password", thePassword)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	if token == nil {
		t.Fatal("No token nor error returned")
	}
	if token.AccessToken != theToken {
		t.Fatalf("Received token: %s, expected: %s", token.AccessToken, theToken)
	}
	if token.RefreshToken != theRefreshToken {
		t.Fatalf("Received token: %s, expected: %s", token.AccessToken, theToken)
	}

	cfg = OAuthClientConfig{
		Client: theClient,
		Endpoint: oAuthEndpointConfig{
			AuthURL:  as.URL,
			TokenURL: ts.URL,
		},
		RedirectURL: rs.URL,
		Secret:      theSecret,
	}
	token, err = cfg.GetCodeAuthToken("guest_access", "1")
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	if token == nil {
		t.Fatal("No token nor error returned")
	}
	if token.AccessToken != theToken {
		t.Fatalf("Received token: %s, expected: %s", token.AccessToken, theToken)
	}
	if token.RefreshToken != theRefreshToken {
		t.Fatalf("Received token: %s, expected: %s", token.AccessToken, theToken)
	}
}

func TestGetAuthToken(t *testing.T) {
	theToken := "thetoken"
	theRefreshToken := "therefreshtoken"
	theUsername := "theuser"
	thePassword := "thepassword"
	theClient := "my_client"
	rs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer rs.Close()
	as := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		qv := r.URL.Query()
		clientID := qv.Get("client_id")
		guestAccess := qv.Get("guest_access")
		username := qv.Get("username")
		password := qv.Get("password")
		if clientID != theClient {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if guestAccess == "" && (username != theUsername || password != thePassword) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		rdURL := qv.Get("redirect_uri")
		rURL := fmt.Sprintf(`%s#access_token=%s&refresh_token=%s`, rdURL, theToken, theRefreshToken)
		http.Redirect(w, r, rURL, http.StatusPermanentRedirect)
	}))
	defer as.Close()

	cfg := OAuthClientConfig{
		Client: "nobody",
		Endpoint: oAuthEndpointConfig{
			AuthURL: as.URL,
		},
		RedirectURL: rs.URL,
	}
	_, err := cfg.GetAuthToken()
	go_test_utils.ExpectError(t, err, "Expected error")

	cfg = OAuthClientConfig{
		Client:      theClient,
		Endpoint:    oAuthEndpointConfig{},
		RedirectURL: rs.URL,
	}
	_, err = cfg.GetAuthToken("username", theUsername, "password", thePassword)
	go_test_utils.ExpectError(t, err, "Expected error")

	cfg = OAuthClientConfig{
		Client: theClient,
		Endpoint: oAuthEndpointConfig{
			AuthURL: as.URL,
		},
		RedirectURL: rs.URL,
	}
	_, err = cfg.GetAuthToken("username", theUsername, "password", "wrong")
	go_test_utils.ExpectError(t, err, "Expected error")

	cfg = OAuthClientConfig{
		Client: theClient,
		Endpoint: oAuthEndpointConfig{
			AuthURL: as.URL,
		},
		RedirectURL: rs.URL,
	}
	token, err := cfg.GetAuthToken("username", theUsername, "password", thePassword)
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	if token == nil {
		t.Fatal("No token nor error returned")
	}
	if token.AccessToken != theToken {
		t.Fatalf("Received token: %s, expected: %s", token.AccessToken, theToken)
	}
	if token.RefreshToken != theRefreshToken {
		t.Fatalf("Received token: %s, expected: %s", token.AccessToken, theToken)
	}

	cfg = OAuthClientConfig{
		Client: theClient,
		Endpoint: oAuthEndpointConfig{
			AuthURL: as.URL,
		},
		RedirectURL: rs.URL,
	}
	token, err = cfg.GetAuthToken("guest_access", "1")
	go_test_utils.ExpectNoError(t, err, errorStringIfNotNil(err))
	if token == nil {
		t.Fatal("No token nor error returned")
	}
	if token.AccessToken != theToken {
		t.Fatalf("Received token: %s, expected: %s", token.AccessToken, theToken)
	}
	if token.RefreshToken != theRefreshToken {
		t.Fatalf("Received token: %s, expected: %s", token.AccessToken, theToken)
	}
}
