package app_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	otelObs "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	mopsos "github.com/adfinis-sygroup/mopsos/app"
	"github.com/adfinis-sygroup/mopsos/app/db"
	"github.com/adfinis-sygroup/mopsos/app/models"
)

func eventStub(record *models.Record) models.EventData {
	ctx := context.Background()

	evt := cloudevents.NewEvent(cloudevents.VersionV1)
	evt.SetType("com.example.record")

	err := evt.SetData("application/json", record)
	if err != nil {
		panic(err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.NeverSample()),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("test"),
		)),
	)

	otel.SetTracerProvider(tp)
	tracer := otel.Tracer("")
	ctx, span := tracer.Start(ctx, "")

	defer span.End()

	otelObs.InjectDistributedTracingExtension(ctx, evt)
	fmt.Printf("%+v\n", ctx)
	fmt.Printf("%+v\n", evt)

	return models.EventData{
		Event:  evt,
		Record: *record,
	}
}

func Test_Handler_HandleEvent(t *testing.T) {
	type args struct {
		eventData models.EventData
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "simple event with minimal data",
			args: args{
				eventData: eventStub(
					&models.Record{
						ClusterName:        "test",
						ApplicationName:    "test",
						ApplicationVersion: "test",
					},
				),
			},
		},
		{
			name: "event with complete data",
			args: args{
				eventData: eventStub(
					&models.Record{
						ClusterName:         "cluster-name",
						InstanceId:          "cluster-instance",
						ApplicationName:     "app-name",
						ApplicationInstance: "app-instance",
						ApplicationVersion:  "app-version",
					},
				),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := tt.args.eventData.Event
			evtRecord := &models.Record{}
			err := evt.DataAs(evtRecord)
			if err != nil {
				t.Errorf("failed to unmarshal event data: %v", err)
			}

			gdb, err := db.NewDBConnection(&mopsos.Config{
				DBProvider:    "sqlite",
				DBDSN:         "file::memory:?cache=shared",
				DBMigrate:     true,
				EnableTracing: false,
			})
			if err != nil {
				t.Errorf("failed to connect to database: %v", err)
			}

			h := mopsos.NewHandler(true, gdb)

			if err := h.HandleEvent(tt.args.eventData); (err != nil) != tt.wantErr {
				t.Errorf("Handler.HandleEvent() error = %v, wantErr %v", err, tt.wantErr)
			}

			dbRecord := &models.Record{}
			gdb.Last(dbRecord)
			if gdb.Error != nil {
				t.Fatalf("an error '%s' was not expected when querying the database", gdb.Error)
			}
			// copy generated values to the original struct for comparison
			evtRecord.CreatedAt = dbRecord.CreatedAt
			evtRecord.UpdatedAt = dbRecord.UpdatedAt
			evtRecord.DeletedAt = dbRecord.DeletedAt
			evtRecord.ID = dbRecord.ID
			if reflect.DeepEqual(dbRecord, evtRecord) == false {
				t.Errorf("Handler.HandleEvent() = %v, want %v", dbRecord, evtRecord)
			}
		})
	}

}
