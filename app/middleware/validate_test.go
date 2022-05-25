package middleware_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	httproto "github.com/cloudevents/sdk-go/v2/protocol/http"

	"github.com/adfinis-sygroup/mopsos/app/middleware"
	"github.com/adfinis-sygroup/mopsos/app/types"
)

func Test_Validate(t *testing.T) {
	handlerCalled := false
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event := r.Context().Value(types.ContextEvent).(*cloudevents.Event)

		if event == nil {
			t.Errorf("event should be stored in the req context")
		}
		handlerCalled = true

	})

	body := []byte(`{"specversion":"1.0","type":"test","datacontenttype": "application/json","data": {"cluster_name": "username"}}`)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/webhook", bytes.NewReader(body))
	req.Header.Add("Content-Type", "application/cloudevents+json")

	res := httptest.NewRecorder()

	// get event
	message := httproto.NewMessageFromHttpRequest(req)
	event, _ := binding.ToEvent(req.Context(), message)

	ctx := context.WithValue(req.Context(), types.ContextEvent, event)
	ctx = context.WithValue(ctx, types.ContextUsername, "username")

	load := middleware.Validate(handler)
	load.ServeHTTP(res, req.WithContext(ctx))

	req.Body.Close()

	if !handlerCalled {
		t.Errorf("handler should have been called")
	}
}

func Test_ValidateInvalid(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not have been called")
	})

	body := []byte(`{"specversion":"1.0","type":"test","datacontenttype": "application/json","data": {"cluster_name": "invalid"}}`)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/webhook", bytes.NewReader(body))
	req.Header.Add("Content-Type", "application/cloudevents+json")

	res := httptest.NewRecorder()

	// get event
	message := httproto.NewMessageFromHttpRequest(req)
	event, _ := binding.ToEvent(req.Context(), message)

	ctx := context.WithValue(req.Context(), types.ContextEvent, event)
	ctx = context.WithValue(ctx, types.ContextUsername, "username")

	load := middleware.Validate(handler)
	load.ServeHTTP(res, req.WithContext(ctx))

	req.Body.Close()

	if res.Code != http.StatusUnauthorized {
		t.Errorf("expected unauthorized status code, got %d", res.Code)
	}
}

func Test_ValidateFailedMarshal(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not have been called")
	})

	body := []byte(`{"specversion":"1.0","type":"test","datacontenttype": "test/plain","data": "this is not json"}`)
	req := httptest.NewRequest(http.MethodPost, "http://example.com/webhook", bytes.NewReader(body))
	req.Header.Add("Content-Type", "application/cloudevents+json")

	res := httptest.NewRecorder()

	// get event
	message := httproto.NewMessageFromHttpRequest(req)
	event, _ := binding.ToEvent(req.Context(), message)

	ctx := context.WithValue(req.Context(), types.ContextEvent, event)
	ctx = context.WithValue(ctx, types.ContextUsername, "username")

	load := middleware.Validate(handler)
	load.ServeHTTP(res, req.WithContext(ctx))

	req.Body.Close()

	if res.Code != http.StatusInternalServerError {
		t.Errorf("expected internal server error status code, got %d", res.Code)
	}
}
