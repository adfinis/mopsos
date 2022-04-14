package app

import (
	"context"

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
func (h *Handler) HandleEvents(eventChan chan models.EventData) error {
	// block on the event channel while ranging over its contents
	for data := range eventChan {
		err := h.HandleEvent(data)
		if err != nil {
			logrus.WithField("event", data.Event).WithError(err).Error("failed to handle event")
		}
	}
	return nil
}

func (h *Handler) HandleEvent(data models.EventData) error {
	log := logrus.WithField("event", data.Event)
	log.Debug("received event")

	ctx := context.Background()

	log.WithField("record", data.Record).Debug("creating record")

	h.database.WithContext(ctx).Clauses(
		clause.OnConflict{
			Columns: []clause.Column{
				{Name: "cluster_name"},
				{Name: "instance_id"},
				{Name: "application_name"},
				{Name: "application_instance"},
			},
			UpdateAll: true,
		},
	).Create(&data.Record)

	return nil
}
