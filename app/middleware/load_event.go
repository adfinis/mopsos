package middleware

import (
	"context"
	"net/http"

	otelObs "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	"github.com/cloudevents/sdk-go/v2/binding"
	httproto "github.com/cloudevents/sdk-go/v2/protocol/http"
	"github.com/sirupsen/logrus"

	"github.com/adfinis-sygroup/mopsos/app/types"
)

// LoadEvent middleware loads event from the request
func LoadEvent(next http.Handler, enableTracing bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get event
		message := httproto.NewMessageFromHttpRequest(r)
		event, err := binding.ToEvent(r.Context(), message)
		if err != nil {
			logrus.WithError(err).Error("failed to decode event")
			return
		}
		if enableTracing {
			// inject the span context into the event so it can be use i.e. while inserting to the database
			otelObs.InjectDistributedTracingExtension(r.Context(), *event)
		}
		logrus.Debugf("received event: %v", event)

		ctx := context.WithValue(r.Context(), types.ContextEvent, event)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
