package middleware_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"

	"github.com/adfinis-sygroup/mopsos/app/middleware"
	"github.com/adfinis-sygroup/mopsos/app/types"
)

func Test_LoadEvent(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event := r.Context().Value(types.ContextEvent).(*cloudevents.Event)
		if event.Type() != "test" {
			t.Errorf("event type should be test")
		}
	})

	body := []byte(`{"specversion":"1.0","type":"test"}`)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/webhook", bytes.NewReader(body))
	req.Header.Add("Content-Type", "application/cloudevents+json")

	res := httptest.NewRecorder()

	load := middleware.LoadEvent(handler, true)
	load.ServeHTTP(res, req)

	req.Body.Close()
}
