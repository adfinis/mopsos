package app

import (
	"errors"

	"github.com/adfinis-sygroup/mopsos/app/models"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// App is the application struct
type App struct {
	Server  *Server
	Handler *Handler
}

// NewApp creates a new App
func NewApp(c *Config, db *gorm.DB) (*App, error) {
	if db == nil {
		return nil, errors.New("database is nil")
	}
	return &App{
		Server:  NewServer(c),
		Handler: NewHandler(c.EnableTracing, db),
	}, nil
}

func (a *App) Run() {
	// eventChan is used to asynchronously pass events receiver from the Server to the Handler
	eventChan := make(chan models.EventData)

	// handle events in background goroutine
	go func() {
		err := a.Handler.HandleEvents(eventChan)
		if err != nil {
			logrus.WithError(err).Error("error handling events")
		}
	}()

	// start blocking server
	a.Server.WithEventChannel(eventChan).Start()
}
