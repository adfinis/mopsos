package app_test

import (
	"testing"

	"gorm.io/gorm"

	mopsos "github.com/adfinis-sygroup/mopsos/app"
)

func Test_NewAppFailWhenNoDatabase(t *testing.T) {
	_, err := mopsos.NewApp(&mopsos.Config{
		HttpListener:  ":8080",
		EnableTracing: false,
		TracingTarget: ""}, nil)
	if err == nil {
		t.Errorf("error: %v", err)
	}
}
func Test_NewApp(t *testing.T) {
	dbMock := &gorm.DB{}
	a, _ := mopsos.NewApp(&mopsos.Config{
		HttpListener:  ":8080",
		EnableTracing: false,
		TracingTarget: ""}, dbMock)
	if a == nil {
		t.Error("NewApp() returned nil")
	}
}
