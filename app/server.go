package app

import (
	"encoding/json"
	"net/http"

	"github.com/cloudevents/sdk-go/v2/event"
	http_logrus "github.com/improbable-eng/go-httpwares/logging/logrus"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

	"github.com/adfinis-sygroup/mopsos/app/middleware"
	"github.com/adfinis-sygroup/mopsos/app/models"
	"github.com/adfinis-sygroup/mopsos/app/types"
)

// Server is the main webserver struct
type Server struct {
	config *Config

	EventChan chan<- models.EventData
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
	mux.HandleFunc("/health", s.HandleHealthCheck)
	mux.Handle("/webhook", otelhttp.NewHandler(
		middleware.Authenticate(
			middleware.LoadEvent(
				middleware.Validate(
					http.HandlerFunc(s.HandleWebhook),
				),
				s.config.EnableTracing,
			),
			s.config.BasicAuthUsers,
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

func (s *Server) HandleHealthCheck(w http.ResponseWriter, r *http.Request) {
	// an example API handler
	err := json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	if err != nil {
		logrus.WithError(err).Error("error encoding response")
	}
}

func (s *Server) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// get middleware data from context
	event := r.Context().Value(types.ContextEvent).(*event.Event)
	record := r.Context().Value(types.ContextRecord).(*models.Record)

	// send the event to the main app via the async channel
	s.EventChan <- models.EventData{
		Event:  *event,
		Record: *record,
	}
	// return 202 accepted once the event is on the queue
	w.WriteHeader(http.StatusAccepted)
}
