package app_test

import (
	"context"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go/v2"
	"gorm.io/gorm"

	mopsos "github.com/adfinis-sygroup/mopsos/app"
)

func Test_ServerHandleReceivedEvent(t *testing.T) {
	dbMock := &gorm.DB{}
	a, _ := mopsos.NewApp(&mopsos.Config{
		HttpListener:  ":8080",
		EnableTracing: false,
		TracingTarget: "",
	}, dbMock)
	eventChan := make(chan cloudevents.Event)
	a.Server.WithEventChannel(eventChan)
	ctx := context.TODO()
	mockEvent := cloudevents.NewEvent(
		cloudevents.VersionV1,
	)
	mockEvent.SetType("test")
	go func() {
		err := a.Server.HandleReceivedEvent(ctx, mockEvent)
		if err != nil {
			t.Errorf("error: %v", err)
		}
	}()

	for i := range eventChan {
		if i.Type() != "test" {
			t.Errorf("error: %v", i.Type())
		}
		close(eventChan)
	}
}

func Test_ServerWithEventChannel(t *testing.T) {
	dbMock := &gorm.DB{}
	a, _ := mopsos.NewApp(&mopsos.Config{
		HttpListener:  ":8080",
		EnableTracing: false,
	}, dbMock)
	eventChan := make(chan cloudevents.Event)
	a.Server.WithEventChannel(eventChan)
	if a.Server.EventChan != eventChan {
		t.Errorf("error: %v", "event channel not set")
	}
}
