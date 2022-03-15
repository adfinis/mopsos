# Mopsos

Receives CloudEvents from Argo CD Notifications and stores them for later analysis.

Mopsos knows what application is installed to which cluster and version it helps see
what you need to update.

## Usage

```console
Mopsos receives events and stores them in a database for later analysis.

Usage:
  mopsos [flags]

Flags:
      --db-dsn string           Database DSN (default "file::memory:?cache=shared")
      --db-migrate              Migrate database schema on startup (default true)
      --db-provider string      Database provider, either 'sqlite' or 'postgres' (default "sqlite")
      --debug                   Enable debug mode
  -h, --help                    help for mopsos
      --http-listener string    HTTP listener (default ":8080")
      --otel                    Enable OpenTelemetry tracing
      --otel-collector string   Endpoint for OpenTelemetry Collector. On a local cluster the collector should be accessible through a NodePort service at the localhost:30078 endpoint. Otherwise replace localhost with the collector endpoint. (default "localhost:30079")
```

## Development

```bash
# run tests (./... recursivley scans the repo for _test.go files)
go test ./...

# generate and view test coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```
