package app

import (
	"context"
	"encoding/json"
	"net/http"

	otelObs "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/cloudevents/sdk-go/v2/binding"
	"github.com/cloudevents/sdk-go/v2/protocol"
	httproto "github.com/cloudevents/sdk-go/v2/protocol/http"
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
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		// an example API handler
		err := json.NewEncoder(w).Encode(map[string]bool{"ok": true})
		if err != nil {
			logrus.WithError(err).Error("error encoding response")
		}
	})
	http.Handle("/webhook", otelhttp.NewHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		// get event
		message := httproto.NewMessageFromHttpRequest(r)
		event, err := binding.ToEvent(context.TODO(), message)
		if err != nil {
			logrus.WithError(err).Error("failed to decode event")
			return
		}
		err = s.HandleReceivedEvent(ctx, *event)
		if err != nil {
			logrus.WithError(err).Error("failed to handle event")
			return
		}
	}), "webhook-receiver"))

	logrus.WithField("listener", s.config.HttpListener).Info("Starting server")
	logrus.Fatal(http.ListenAndServe(s.config.HttpListener, nil))
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
