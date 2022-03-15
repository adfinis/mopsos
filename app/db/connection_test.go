package db_test

import (
	"testing"

	mopsos "github.com/adfinis-sygroup/mopsos/app"
	"github.com/adfinis-sygroup/mopsos/app/db"
)

func Test_NewDBConnection(t *testing.T) {
	tests := []struct {
		name    string
		config  *mopsos.Config
		wantErr bool
	}{
		{
			name: "sqlite in memory connection (use for e2e testing)",
			config: &mopsos.Config{
				DBProvider:    "sqlite",
				DBDSN:         "file::memory:?cache=shared",
				DBMigrate:     true,
				EnableTracing: true,
			},
			wantErr: false,
		},
		{
			name: "connect to postgres",
			config: &mopsos.Config{
				DBProvider:    "postgres",
				DBDSN:         "postgres://postgres:postgres@localhost:5432/mopsos?sslmode=disable",
				DBMigrate:     true,
				EnableTracing: false,
			},
			// there is no postgres database so we fail
			wantErr: true,
		},
		{
			name: "connection with invalid provider",
			config: &mopsos.Config{
				DBProvider:    "invalid",
				DBDSN:         "invalid",
				DBMigrate:     false,
				EnableTracing: false,
			},
			// we should always fail when a non supported provider is used
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := db.NewDBConnection(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDBConnection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
