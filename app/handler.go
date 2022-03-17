package app

import (
	"context"

	otelObs "github.com/cloudevents/sdk-go/observability/opentelemetry/v2/client"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/adfinis-sygroup/mopsos/app/models"
)

type Handler struct {
	database *gorm.DB

	enableTracing bool
}

func NewHandler(enableTracing bool, db *gorm.DB) *Handler {
	return &Handler{
		database:      db,
		enableTracing: enableTracing,
	}
}

// HandleEvents blocks on the queue and handles events
func (h *Handler) HandleEvents(eventChan chan cloudevents.Event) error {
	// block on the event channel while ranging over its contents
	for event := range eventChan {
		err := h.HandleEvent(event)
		if err != nil {
			logrus.WithField("event", event).WithError(err).Error("failed to handle event")
		}
	}
	return nil
}

func (h *Handler) HandleEvent(event cloudevents.Event) error {
	log := logrus.WithField("event", event)
	log.Debug("received event")

	ctx := context.Background()

	if h.enableTracing {
		ctx = otelObs.ExtractDistributedTracingExtension(ctx, event)
	}

	record := &models.Record{}

	err := event.DataAs(record)
	if err != nil {
		log.WithError(err).Errorf("failed to unmarshal event data")
		return err
	}

	log.WithField("record", record).Debug("creating record")

	h.database.WithContext(ctx).Clauses(
		clause.OnConflict{
			Columns: []clause.Column{
				clause.Column{Name: "cluster_name"},
				clause.Column{Name: "instance_id"},
				clause.Column{Name: "application_name"},
				clause.Column{Name: "application_instance"},
			},
			UpdateAll: true,
		},
	).Create(record)

	return nil
}
