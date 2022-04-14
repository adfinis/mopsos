package models

import cloudevents "github.com/cloudevents/sdk-go/v2"

// EventData is the data structure for passing events between the server and the handler
type EventData struct {
	Event  cloudevents.Event
	Record Record
}
