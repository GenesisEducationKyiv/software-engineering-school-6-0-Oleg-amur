# Testing

The project uses regular Go unit tests, Docker-backed integration tests, and Playwright E2E tests.

The `Makefile` commands below are the preferred local entry points. If `make` is unavailable, run the direct commands shown in each section.

## Run All Tests

```sh
make test
```

Without `make`:

```sh
sh -c 'cd shared/contracts && go test -v ./... && cd ../../services/subscription-service && go test -v ./... && go test -tags=integration -count=1 -v ./test/integration/... && cd ../notification-worker && go test -v ./... && cd ../.. && docker compose -f test/e2e/docker-compose-e2e-tests.yaml up --build --abort-on-container-exit --exit-code-from playwright; code=$?; docker compose -f test/e2e/docker-compose-e2e-tests.yaml down -v --remove-orphans; exit $code'
```

## Unit Tests

```sh
make test-unit
```

Without `make`:

```sh
cd shared/contracts && go test -v ./...
cd services/subscription-service && go test -v ./...
cd services/notification-worker && go test -v ./...
cd test/e2e/fakes/github && go test -v ./...
```

Unit tests live next to the code they cover, for example:

- `services/subscription-service/internal/service/*_test.go`
- `services/subscription-service/internal/scanner/*_test.go`
- `services/notification-worker/internal/service/*_test.go`

## Integration Tests

```sh
make test-integration
```

Without `make`:

```sh
cd services/subscription-service && go test -tags=integration -count=1 -v ./test/integration/...
```

Integration tests live under `services/subscription-service/test/integration`:

- `http` covers HTTP API endpoint behavior.
- `grpc` covers gRPC API endpoint behavior.
- `repository` covers PostgreSQL repository behavior.
- `testkit` contains shared setup helpers, Docker Postgres startup, migrations, database cleanup, and controllable external fakes.

Docker must be available. The tests start PostgreSQL automatically from scratch with Testcontainers and reset database state between tests.

## E2E Tests

```sh
make test-e2e
```

Without `make`:

```sh
sh -c 'docker compose -f test/e2e/docker-compose-e2e-tests.yaml up --build --abort-on-container-exit --exit-code-from playwright; code=$?; docker compose -f test/e2e/docker-compose-e2e-tests.yaml down -v --remove-orphans; exit $code'
```

E2E tests live under `test/e2e` because they cover the whole deployed system. Docker Compose starts PostgreSQL, Mailpit, RabbitMQ, a controlled fake GitHub HTTP service, the main service, the notification worker, and a Playwright runner.

## Coverage

```sh
make coverage
```

Without `make`:

```sh
make coverage
```

Unit and integration coverage profiles are written to `coverage/`. CI uploads those profiles as workflow artifacts.

## Naming Conventions

Go test functions use the `TestType_Method` pattern for unit tests, for example `TestRepositoryService_GetOrCreate`. More specific scenarios can append behavior after another underscore, for example `TestNotificationService_ProcessReleaseEvent_BuildsReleaseMessage`.

Integration tests live under `services/subscription-service/test/integration` and use the `integration` build tag, so test names do not repeat `Integration`. Prefer names like `TestHTTPSubscribe_CreatesPendingSubscription`, `TestGRPCSubscribe_CreatesPendingSubscription`, and `TestSubscriptionRepository`.

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

## Architecture Lint

Architecture dependency rules are checked by `go-arch-lint` and run as part of `make lint`.

```sh
go install github.com/fe3dback/go-arch-lint@v1.15.0
make arch-lint
```

Each service owns its `.go-arch-lint.yml`. The configs check that domain, use cases, workflows, transport, persistence, and adapters keep the intended dependency direction inside `subscription-service`, `release-tracker`, and `notification-worker`.

Vendor and shared-module imports are controlled per layer: outer layers such as transports, adapters, persistence, and composition roots may use vendor packages freely, while domain packages cannot use vendor packages and application-core packages only allow the shared event contracts they publish or consume. `deepScan` remains disabled because the composition roots in `cmd/server` intentionally inject concrete adapters into use cases and workflows; import rules still enforce the package-level dependency boundaries.
