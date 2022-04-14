package models

import (
	"time"

	"gorm.io/gorm"
)

/**
 * Record is the model for the records table
 *
 * This is the main datakeeping model for mopsus. It stores the
 * app info from received events.
 */
type Record struct {
	// gorm.Model values but with json:"-" to hide from output
	ID        uint           `gorm:"primarykey" json:"-"`
	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`

	// actual payload of record
	ClusterName         string `json:"cluster_name" gorm:"uniqueIndex:idx_unique"`
	InstanceId          string `json:"instance_id" gorm:"uniqueIndex:idx_unique"`
	ApplicationName     string `json:"application_name" gorm:"uniqueIndex:idx_unique"`
	ApplicationInstance string `json:"application_instance" gorm:"uniqueIndex:idx_unique"`
	ApplicationVersion  string `json:"application_version" gorm:"not null"`
}

var ContextUsername string = "username"
