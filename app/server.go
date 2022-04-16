package app

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/adfinis-sygroup/mopsos/app/models"
	otelObs "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/event"
	httproto "github.com/cloudevents/sdk-go/v2/protocol/http"
	http_logrus "github.com/improbable-eng/go-httpwares/logging/logrus"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Server is the main webserver struct
type Server struct {
	config *Config

	EventChan chan<- models.EventData
}

type eventContext string

var ContextUsername eventContext = "mopsos.username"
var ContextEvent eventContext = "mopsos.event"
var ContextRecord eventContext = "mopsos.record"

// NewServer creates a server that receives CloudEvents from the network
func NewServer(cfg *Config) *Server {
	return &Server{
		config: cfg,
	}
}

// Start starts the server and listens for incoming events
func (s *Server) Start() {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.HandleHealthCheck)
	mux.Handle("/webhook", otelhttp.NewHandler(
		s.Authenticate(
			s.LoadEvent(
				s.Validate(
					http.HandlerFunc(s.HandleWebhook),
				),
			),
		),
		"webhook-receiver"),
	)

	logrus.WithField("listener", s.config.HttpListener).Info("Starting server")
	loggingMiddleware := http_logrus.Middleware(
		logrus.WithFields(logrus.Fields{}),
	)(mux)
	logrus.Fatal(http.ListenAndServe(s.config.HttpListener, loggingMiddleware))
}

// WithEventChannel sets the event channel for the server
func (s *Server) WithEventChannel(eventChan chan<- models.EventData) *Server {
	s.EventChan = eventChan
	return s
}

// Authenticate middleware handles checking credentials
func (s *Server) Authenticate(next http.Handler) http.Handler {
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
		if s.config.BasicAuthUsers[username] != password {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		ctx := context.WithValue(r.Context(), ContextUsername, username)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// LoadEvent middlerware loads event from the request
func (s *Server) LoadEvent(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// get event
		message := httproto.NewMessageFromHttpRequest(r)
		event, err := binding.ToEvent(r.Context(), message)
		if err != nil {
			logrus.WithError(err).Error("failed to decode event")
			return
		}
		if s.config.EnableTracing {
			// inject the span context into the event so it can be use i.e. while inserting to the database
			otelObs.InjectDistributedTracingExtension(r.Context(), *event)
		}
		logrus.Debugf("received event: %v", event)

		ctx := context.WithValue(r.Context(), ContextEvent, event)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Validate middleware handles checking received events for validity
func (s *Server) Validate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		event := r.Context().Value(ContextEvent).(event.Event)
		record := &models.Record{}

		if err := event.DataAs(record); err != nil {
			logrus.WithError(err).Errorf("failed to unmarshal event data")
			http.Error(w, "failed to unmarshal event data", http.StatusInternalServerError)
			return
		}

		// reject record that have not been sent from the right auth
		if record.ClusterName != r.Context().Value(ContextUsername).(string) {
			http.Error(w, "event data does not match username", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ContextRecord, record)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// an example API handler
	json.NewEncoder(w).Encode(map[string]bool{"ok": true})
}

func (s *Server) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// get middleware data from context
	event := r.Context().Value(ContextEvent).(event.Event)
	record := r.Context().Value(ContextRecord).(*models.Record)

	// send the event to the main app via the async channel
	s.EventChan <- models.EventData{
		Event:  event,
		Record: *record,
	}
	// return 202 accepted once the event is on the queue
	w.WriteHeader(http.StatusAccepted)
}
