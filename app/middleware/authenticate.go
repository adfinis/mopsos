package middleware

import (
	"context"
	"net/http"

	"github.com/sirupsen/logrus"

	"github.com/adfinis-sygroup/mopsos/app/types"
)

// Authenticate middleware handles checking credentials
func Authenticate(next http.Handler, basicAuthUsers map[string]string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get basic auth credentials
		username, password, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}

		logrus.WithFields(logrus.Fields{
			"username": username,
		}).Debug("checking credentials")
		if basicAuthUsers[username] != password {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), types.ContextUsername, username)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
