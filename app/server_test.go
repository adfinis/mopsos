package app_test

import (
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

func Test_ServerHandleReceivedEvent(t *testing.T) {
	a, _, eventChan := newApp()

	mockEvent := cloudevents.NewEvent(
		cloudevents.VersionV1,
	)
	a.Server.ReceivedEvent = mockEvent
	a.Server.ReceivedRecord = &models.Record{}
	mockEvent.SetType("test")
	go func() {
		err := a.Server.HandleReceivedEvent()
		if err != nil {
			t.Errorf("error: %v", err)
		}
	}()

	for data := range eventChan {
		i := data.Event
		if i.Type() != "test" {
			t.Errorf("error: %v", i.Type())
		}
		close(a.Server.EventChan)
	}
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

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.SetBasicAuth("username", "password")

	res := httptest.NewRecorder()

	if a.Server.AuthenticatedUser != "" {
		t.Errorf("authenticated user should be empty")
	}

	auth := a.Server.Authenticate(handler)
	auth.ServeHTTP(res, req)

	if a.Server.AuthenticatedUser != "username" {
		t.Errorf("username not set")
	}
}
