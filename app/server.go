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

	EventChan chan<- models.EventData

	AuthenticatedUser string
	ReceivedEvent     cloudevents.Event
	ReceivedRecord    *models.Record
}

type eventContext string

var contextUsername eventContext = "mopsos.username"

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
	mux.Handle("/webhook", otelhttp.NewHandler(
		s.Authenticate(
			s.LoadEvent(
				s.Validate(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					err := s.HandleReceivedEvent()
					if err != nil {
						logrus.WithError(err).Error("failed to handle event")
						return
					}
					// return 202 accepted once the event is on the queue
					w.WriteHeader(http.StatusAccepted)
				}),
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
		if !s.checkAuth(username, password) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		s.AuthenticatedUser = username
		next.ServeHTTP(w, r)
	})
}

// LoadEvent middlerware loads event from the request
func (s *Server) LoadEvent(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), contextUsername, s.AuthenticatedUser)

		// get event
		message := httproto.NewMessageFromHttpRequest(r)
		event, err := binding.ToEvent(ctx, message)
		if err != nil {
			logrus.WithError(err).Error("failed to decode event")
			return
		}
		if s.config.EnableTracing {
			// inject the span context into the event so it can be use i.e. while inserting to the database
			otelObs.InjectDistributedTracingExtension(ctx, *event)
		}

		logrus.Debugf("received event: %v", event)
		s.ReceivedEvent = *event
		next.ServeHTTP(w, r)
	})
}

// Validate middleware handles checking received events for validity
func (s *Server) Validate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO consider how to harmonise this with what the handler does later on
		record := &models.Record{}
		if err := s.ReceivedEvent.DataAs(record); err != nil {
			logrus.WithError(err).Errorf("failed to unmarshal event data")
			http.Error(w, "failed to unmarshal event data", http.StatusInternalServerError)
			return
		}

		// reject record that have not been sent from the right auth
		if record.ClusterName != s.AuthenticatedUser {
			http.Error(w, "event data does not match username", http.StatusUnauthorized)
			return
		}

		s.ReceivedRecord = record
		next.ServeHTTP(w, r)
	})
}

// HandleReceivedEvent is the handler for the cloudevents receiver, public for testing
func (s *Server) HandleReceivedEvent() protocol.Result {

	// send the event to the main app via the async channel
	s.EventChan <- models.EventData{
		Event:  s.ReceivedEvent,
		Record: *s.ReceivedRecord,
	}

	return nil
}

// checkAuth checks if the username and password are correct
func (s *Server) checkAuth(username, password string) bool {
	logrus.WithFields(logrus.Fields{
		"username": username,
	}).Debug("checking credentials")
	return s.config.BasicAuthUsers[username] == password
}
