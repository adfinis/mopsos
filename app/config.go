package app

// Config type for config
type Config struct {
	DBProvider string
	DBDSN      string
	DBMigrate  bool

	HttpListener   string
	BasicAuthUsers map[string]string

	EnableTracing bool
	TracingTarget string
}
