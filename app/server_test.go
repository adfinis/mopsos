package app_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
