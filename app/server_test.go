package app_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"gorm.io/gorm"

	mopsos "github.com/adfinis-sygroup/mopsos/app"
	"github.com/adfinis-sygroup/mopsos/app/models"
)

func newApp() (*mopsos.App, *gorm.DB, chan models.EventData) {
	dbMock := &gorm.DB{}
	a, _ := mopsos.NewApp(&mopsos.Config{
		HttpListener:  ":8080",
		EnableTracing: false,
		TracingTarget: "",

		BasicAuthUsers: map[string]string{
			"username": "password",
		},
	}, dbMock)

	eventChan := make(chan models.EventData)
	a.Server.WithEventChannel(eventChan)

	return a, dbMock, eventChan
}

func Test_ServerWithEventChannel(t *testing.T) {
	a, _, eventChan := newApp()

	a.Server.WithEventChannel(eventChan)
	if a.Server.EventChan != eventChan {
		t.Errorf("event channel not set")
	}
}

func Test_Authenticate(t *testing.T) {
	a, _, _ := newApp()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(mopsos.ContextUsername) != "username" {
			t.Error("username not set")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.SetBasicAuth("username", "password")

	res := httptest.NewRecorder()

	auth := a.Server.Authenticate(handler)
	auth.ServeHTTP(res, req)

	if res.Result().StatusCode != http.StatusOK {
		t.Errorf("status code should be 200")
	}

}

func Test_AuthenticateNoHeader(t *testing.T) {
	a, _, _ := newApp()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

	res := httptest.NewRecorder()

	auth := a.Server.Authenticate(handler)
	auth.ServeHTTP(res, req)

	if res.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("status code should be 401")
	}
}

func Test_AuthenticateInvalidUser(t *testing.T) {
	a, _, _ := newApp()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.SetBasicAuth("username", "invalid")

	res := httptest.NewRecorder()

	auth := a.Server.Authenticate(handler)
	auth.ServeHTTP(res, req)

	if res.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("status code should be 401")
	}
}

func Test_LoadEvent(t *testing.T) {
	a, _, _ := newApp()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event := r.Context().Value(mopsos.ContextEvent).(*cloudevents.Event)
		if event.Type() != "test" {
			t.Errorf("event type should be test")
		}
	})

	body := []byte(`{"specversion":"1.0","type":"test"}`)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/webhook", bytes.NewReader(body))
	req.Header.Add("Content-Type", "application/cloudevents+json")

	res := httptest.NewRecorder()

	load := a.Server.LoadEvent(handler)
	load.ServeHTTP(res, req)

	req.Body.Close()
}

func Test_HandleHealthCheck(t *testing.T) {
	a, _, _ := newApp()

	req := httptest.NewRequest(http.MethodGet, "http://example.com/health", nil)
	res := httptest.NewRecorder()

	a.Server.HandleHealthCheck(res, req)

	if res.Result().StatusCode != http.StatusOK {
		t.Errorf("status code should be 200")
	}
	expected := "{\"ok\":true}\n"
	if res.Body.String() != expected {
		t.Errorf("body should be %v got %s", expected, res.Body.String())
	}
}
