package app_test

import (
	"testing"

	"github.com/adfinis-sygroup/mopsos/app"
	"gorm.io/gorm"
)

func Test_NewAppFailWhenNoDatabase(t *testing.T) {
	_, err := app.NewApp(&app.Config{
		HttpListener:  ":8080",
		EnableTracing: false,
		TracingTarget: ""}, nil)
	if err == nil {
		t.Errorf("error: %v", err)
	}
}
func Test_NewApp(t *testing.T) {
	dbMock := &gorm.DB{}
	a, _ := app.NewApp(&app.Config{
		HttpListener:  ":8080",
		EnableTracing: false,
		TracingTarget: ""}, dbMock)
	if a == nil {
		t.Error("NewApp() returned nil")
	}
}
