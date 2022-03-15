package db

import (
	"github.com/adfinis-sygroup/mopsos/app"
	"github.com/adfinis-sygroup/mopsos/app/models"
	gorm_logrus "github.com/onrik/gorm-logrus"
	"github.com/uptrace/opentelemetry-go-extra/otelgorm"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// NewDBConnection creates a new database connection for a specific provider
func NewDBConnection(config *app.Config) (*gorm.DB, error) {
	var dialector gorm.Dialector

	switch config.DBProvider {
	case "sqlite":
		dialector = sqlite.Open(config.DBDSN)
	case "postgres":
		dialector = postgres.Open(config.DBDSN)
	default:
		return nil, gorm.ErrNotImplemented
	}

	dbConn, err := gorm.Open(dialector, &gorm.Config{
		Logger: gorm_logrus.New(),
	})
	if err != nil {
		return nil, err
	}
	if config.EnableTracing {
		if err := dbConn.Use(otelgorm.NewPlugin()); err != nil {
			return nil, err
		}
	}
	if config.DBMigrate {
		if err := dbConn.AutoMigrate(&models.Record{}); err != nil {
			return nil, err
		}
	}
	return dbConn, nil
}
