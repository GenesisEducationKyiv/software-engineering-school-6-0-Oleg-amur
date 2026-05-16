# Testing

The project uses regular Go unit tests, Docker-backed integration tests, and Playwright E2E tests.

## Run All Tests

```sh
sh -c 'go test -v ./... && go test -tags=integration -v ./test/integration/... && docker compose -f test/e2e/docker-compose-e2e-tests.yaml up --build --abort-on-container-exit --exit-code-from playwright; code=$?; docker compose -f test/e2e/docker-compose-e2e-tests.yaml down -v --remove-orphans; exit $code'
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

```sh
sh -c 'docker compose -f test/e2e/docker-compose-e2e-tests.yaml up --build --abort-on-container-exit --exit-code-from playwright; code=$?; docker compose -f test/e2e/docker-compose-e2e-tests.yaml down -v --remove-orphans; exit $code'
```

E2E tests live under `test/e2e`. Docker Compose starts PostgreSQL, Mailpit, a controlled fake GitHub HTTP service, the application, and a Playwright runner.

## Naming Conventions

Go test functions use the `TestType_Method` pattern for unit tests, for example `TestRepositoryService_GetOrCreate`. More specific scenarios can append behavior after another underscore, for example `TestNotificationService_ProcessReleaseEvent_BuildsReleaseMessage`.

Integration tests live under `test/integration` and use the `integration` build tag, so test names do not repeat `Integration`. Prefer names like `TestHTTPSubscriptionFlow`, `TestGRPCSubscribe_CreatesPendingSubscription`, and `TestSubscriptionRepository`.

Table-driven tests use `name` for the case description and `want*` fields for expected values:

```go
tests := []struct {
	name    string
	input   string
	want    string
	wantErr error
}{}
```

Assertions use Go-style `got`/`want` wording, with the actual value first:

```go
if got != want {
	t.Fatalf("got %q, want %q", got, want)
}
```

Use `mock*` for small unit-test doubles and `Fake*` for reusable integration-test fakes.
