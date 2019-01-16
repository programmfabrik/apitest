package api

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/programmfabrik/fylr-apitest/lib/test_utils"
)

func TestNewSessionSucceeds(t *testing.T) {
	server := test_utils.NewTestServer(test_utils.Routes{
		"/api/v1/session": func(w *http.ResponseWriter, r *http.Request) {
			(*w).Write([]byte("{\"token\": \"mock\"}"))
		},
	})
	client := server.Client()
	defer server.Close()
	session, err := NewSession(server.URL, client, nil, &Datastore{})
	test_utils.CheckError(t, err, fmt.Sprintf("error creating session: %s", err))
	test_utils.AssertStringEquals(t, session.token, "mock")
}

func TestNewSessionFails(t *testing.T) {
	server := test_utils.NewTestServer(test_utils.Routes{
		"/api/v1/session": func(w *http.ResponseWriter, r *http.Request) {
			(*w).WriteHeader(http.StatusInternalServerError)
			(*w).Write([]byte("{\"code\": \"error.server.fail\"}"))
		},
	})
	client := server.Client()
	defer server.Close()
	_, err := NewSession(server.URL, client, nil, &Datastore{})
	if err == nil || !strings.Contains(err.Error(), "error.server.fail") {
		t.Errorf("expected error to contain: 'error.server.fail', got %s", err)
	}
}

func TestSessionLoginSuccess(t *testing.T) {
	server := test_utils.NewTestServer(test_utils.Routes{
		"/api/v1/session": func(w *http.ResponseWriter, r *http.Request) {
			(*w).Write([]byte("{\"token\": \"mock\"}"))
		},
		"/api/v1/session/authenticate": func(w *http.ResponseWriter, r *http.Request) {
			(*w).Write([]byte("{\"current_max_event_id\": 2391}"))
		},
	})
	client := server.Client()
	defer server.Close()
	auth := &SessionAuthentication{}
	session, err := NewSession(server.URL, client, auth, &Datastore{})
	test_utils.CheckError(t, err, fmt.Sprintf("error creating session: %s", err))
	test_utils.AssertStringEquals(t, session.token, "mock")
	test_utils.AssertIntEquals(t, session.MaxEventId, 2391)
}

func TestSessionLoginFailsNotAuthenticated(t *testing.T) {
	server := test_utils.NewTestServer(test_utils.Routes{
		"/api/v1/session": func(w *http.ResponseWriter, r *http.Request) {
			(*w).Write([]byte("{\"token\": \"mock\"}"))
		},
		"/api/v1/session/authenticate": func(w *http.ResponseWriter, r *http.Request) {
			(*w).WriteHeader(400)
			(*w).Write([]byte("{\"code\": \"error.something.failed\"}"))
		},
	})
	client := server.Client()
	defer server.Close()
	auth := &SessionAuthentication{}
	_, err := NewSession(server.URL, client, auth, &Datastore{})
	if err == nil || !strings.Contains(err.Error(), "error.something.failed") {
		t.Errorf("expected error to contain: 'error.something.failed', got %s", err)
	}
}

func TestSessionSettingsRequest(t *testing.T) {
	server := test_utils.NewTestServer(test_utils.Routes{
		"/api/v1/settings": func(w *http.ResponseWriter, r *http.Request) {
			(*w).Write([]byte("{\"db-name\": \"mock\"}"))
		},
	})
	client := server.Client()
	defer server.Close()
	session := Session{Store: nil, client: client, serverUrl: server.URL}
	settingsResponse, err := session.SendSettingsRequest()
	test_utils.CheckError(t, err, fmt.Sprintf("error sending settings request: %s", err))
	test_utils.AssertStringEquals(t, settingsResponse.DbName, "mock")
}
