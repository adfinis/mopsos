package app

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/adfinis-sygroup/mopsos/app/models"
	otelObs "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/protocol"
	httproto "github.com/cloudevents/sdk-go/v2/protocol/http"
	http_logrus "github.com/improbable-eng/go-httpwares/logging/logrus"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Server is the main webserver struct
type Server struct {
	config *Config

	EventChan chan<- cloudevents.Event
}

// NewServer creates a server that receives CloudEvents from the network
func NewServer(cfg *Config) *Server {
	return &Server{
		config: cfg,
	}
}

// Start starts the server and listens for incoming events
func (s *Server) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// an example API handler
		err := json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		if err != nil {
			logrus.WithError(err).Error("error encoding response")
		}
	})
	mux.Handle("/webhook", otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// get basic auth credentials
		username, password, ok := r.BasicAuth()
		if !ok {
			http.Error(w, "missing Authorization header", http.StatusUnauthorized)
			return
		}
		if !s.checkAuth(username, password) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}

		// get event
		message := httproto.NewMessageFromHttpRequest(r)
		event, err := binding.ToEvent(context.TODO(), message)
		if err != nil {
			logrus.WithError(err).Error("failed to decode event")
			return
		}

		// TODO consider how to harmonise this with what the handler does later on
		record := &models.Record{}
		if err := event.DataAs(record); err != nil {
			logrus.WithError(err).Errorf("failed to unmarshal event data")
			http.Error(w, "failed to unmarshal event data", http.StatusInternalServerError)
			return
		}

		// reject record that have not been sent from the right auth
		if record.ClusterName != username {
			http.Error(w, "event data does not match username", http.StatusUnauthorized)
			return
		}

		err = s.HandleReceivedEvent(ctx, *event)
		if err != nil {
			logrus.WithError(err).Error("failed to handle event")
			return
		}
		// return 202 accepted once the event is on the queue
		w.WriteHeader(http.StatusAccepted)
	}), "webhook-receiver"))

	logrus.WithField("listener", s.config.HttpListener).Info("Starting server")
	loggingMiddleware := http_logrus.Middleware(
		logrus.WithFields(logrus.Fields{}),
	)(mux)
	logrus.Fatal(http.ListenAndServe(s.config.HttpListener, loggingMiddleware))
}

// WithEventChannel sets the event channel for the server
func (s *Server) WithEventChannel(eventChan chan<- cloudevents.Event) *Server {
	s.EventChan = eventChan
	return s
}

// HandleReceivedEvent is the handler for the cloudevents receiver, public for testing
func (s *Server) HandleReceivedEvent(ctx context.Context, event cloudevents.Event) protocol.Result {

	if s.config.EnableTracing {
		// inject the span context into the event so it can be use i.e. while inserting to the database
		otelObs.InjectDistributedTracingExtension(ctx, event)
	}

	// send the event to the main app via the async channel
	s.EventChan <- event

	logrus.Debugf("received event: %v", event)

	return nil
}

// checkAuth checks if the username and password are correct
func (s *Server) checkAuth(username, password string) bool {
	logrus.WithFields(logrus.Fields{
		"username": username,
	}).Debug("checking credentials")
	return s.config.BasicAuthUsers[username] == password
}
