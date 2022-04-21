package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adfinis-sygroup/mopsos/app/middleware"
	"github.com/adfinis-sygroup/mopsos/app/types"
)

func Test_Authenticate(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Context().Value(types.ContextUsername) != "username" {
			t.Error("username not set")
		}
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.SetBasicAuth("username", "password")

	res := httptest.NewRecorder()

	auth := middleware.Authenticate(handler, map[string]string{"username": "password"})
	auth.ServeHTTP(res, req)

	if res.Result().StatusCode != http.StatusOK {
		t.Errorf("status code should be 200")
	}

}

func Test_AuthenticateNoHeader(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)

	res := httptest.NewRecorder()

	auth := middleware.Authenticate(handler, map[string]string{"username": "password"})
	auth.ServeHTTP(res, req)

	if res.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("status code should be 401")
	}
}

func Test_AuthenticateInvalidUser(t *testing.T) {

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.SetBasicAuth("username", "invalid")

	res := httptest.NewRecorder()

	auth := middleware.Authenticate(handler, map[string]string{"username": "password"})
	auth.ServeHTTP(res, req)

	if res.Result().StatusCode != http.StatusUnauthorized {
		t.Errorf("status code should be 401")
	}
}
