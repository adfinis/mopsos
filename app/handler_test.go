package app_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/adfinis-sygroup/mopsos/app"
	"github.com/adfinis-sygroup/mopsos/app/db"
	"github.com/adfinis-sygroup/mopsos/app/models"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"

	otelObs "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func eventStub(record *models.Record) cloudevents.Event {
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

	return evt
}

func Test_Handler_HandleEvent(t *testing.T) {
	type args struct {
		event cloudevents.Event
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "simple event with minimal data",
			args: args{
				event: eventStub(
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
				event: eventStub(
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
			evt := tt.args.event
			evtRecord := &models.Record{}
			err := evt.DataAs(evtRecord)
			if err != nil {
				t.Errorf("failed to unmarshal event data: %v", err)
			}

			gdb, err := db.NewDBConnection(&app.Config{
				DBProvider:    "sqlite",
				DBDSN:         "file::memory:?cache=shared",
				DBMigrate:     true,
				EnableTracing: false,
			})
			if err != nil {
				t.Errorf("failed to connect to database: %v", err)
			}

			h := app.NewHandler(true, gdb)

			if err := h.HandleEvent(evt); (err != nil) != tt.wantErr {
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
