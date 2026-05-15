# Testing

The project uses regular Go unit tests and Docker-backed integration tests.

## Run All Tests

```sh
go test -v ./... && go test -tags=integration -v ./test/integration/...
```

## Unit Tests

```sh
go test -v ./...
```

Unit tests live next to the code they cover, for example:

- `internal/service/*_test.go`
- `internal/scanner/*_test.go`
- `internal/client/github/*_test.go`

## Integration Tests

```sh
go test -tags=integration -v ./test/integration/...
```

Integration tests live under `test/integration`:

- `http` covers HTTP API endpoint behavior.
- `grpc` covers gRPC API endpoint behavior.
- `repository` covers PostgreSQL repository behavior.
- `testkit` contains shared setup helpers, Docker Postgres startup, migrations, database cleanup, and controllable external fakes.

Docker must be available. The tests start PostgreSQL automatically from scratch with Testcontainers and reset database state between tests.

## E2E Tests

There are no E2E tests yet because the application currently has no UI page for Playwright to drive.
