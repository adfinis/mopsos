package middleware

import (
	"context"
	"net/http"

	"github.com/cloudevents/sdk-go/v2/event"
	"github.com/sirupsen/logrus"

	"github.com/adfinis-sygroup/mopsos/app/models"
	"github.com/adfinis-sygroup/mopsos/app/types"
)

// Validate middleware handles checking received events for validity
func Validate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event := r.Context().Value(types.ContextEvent).(*event.Event)
		record := &models.Record{}

		if err := event.DataAs(record); err != nil {
			logrus.WithError(err).Errorf("failed to unmarshal event data")
			http.Error(w, "failed to unmarshal event data", http.StatusInternalServerError)
			return
		}

		// reject record that have not been sent from the right auth
		if record.ClusterName != r.Context().Value(types.ContextUsername).(string) {
			http.Error(w, "event data does not match username", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), types.ContextRecord, record)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
