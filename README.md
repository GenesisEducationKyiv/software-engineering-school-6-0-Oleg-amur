# GitHub Release Notifier

A Go-based service that monitors GitHub repositories for new releases and notifies subscribers via email. It provides both REST and gRPC interfaces for subscription management.

## Features

- **GitHub Monitoring**: Periodically scans GitHub repositories for new releases.
- **Email Notifications**: Notifies subscribers when a new release is detected.
- **Multi-Protocol Support**:
  - **REST API**: Standard HTTP endpoints for subscription management.
  - **gRPC API**: High-performance interface for service-to-service communication.
- **Database per service**: Subscription and repository tracking data are stored in separate PostgreSQL databases.
- **Monitoring**: Includes structured JSON logs, Elasticsearch/Kibana log search, Prometheus RED metrics, and a Grafana dashboard.
- **Dockerized**: Ready to run with Docker and Docker Compose.

## Documentation

- **[System Design](docs/system-design.md)**: High-level overview of the service architecture, C4 diagrams, and core workflows.
- **[Architectural Decision Records (ADR)](docs/adr/adr-summary.md)**: Detailed rationale for key technical choices.

## Tech Stack

- **Language**: [Go](https://go.dev/) (1.25+)
- **Database**: [PostgreSQL](https://www.postgresql.org/)
- **Communication**: [gRPC](https://grpc.io/), [net/http](https://pkg.go.dev/net/http)
- **Configuration**: [cleanenv](https://github.com/ilyakaznacheev/cleanenv)
- **Metrics**: [Prometheus](https://prometheus.io/)
- **Logs**: [Elasticsearch](https://www.elastic.co/elasticsearch), [Kibana](https://www.elastic.co/kibana), [Filebeat](https://www.elastic.co/beats/filebeat)
- **Dashboards**: [Grafana](https://grafana.com/)
- **Containerization**: [Docker](https://www.docker.com/)

## Getting Started

### Prerequisites

- [Docker](https://www.docker.com/get-started) and [Docker Compose](https://docs.docker.com/compose/install/)
- [Go](https://go.dev/doc/install) (optional, for local development)
- [Buf CLI](https://buf.build/docs/cli/installation/) and the Go protobuf generators (for contract changes)

### Configuration

The service is configured using environment variables or a YAML file. You can find an example configuration in `.env.example`.

Key configuration options:
- `SUBSCRIPTION_SERVICE_HTTP_PORT`/`SUBSCRIPTION_SERVICE_GRPC_PORT`: Public API ports.
- `SUBSCRIPTION_SERVICE_DB_*`: Subscription database credentials, name, and host port.
- `RELEASE_TRACKER_HTTP_PORT`: Internal release tracker HTTP port.
- `RELEASE_TRACKER_DB_*`: Release tracker database credentials, name, and host port.
- `SCAN_INTERVAL`: How often to check for new releases (e.g., `1m`, `1h`).
- `GITHUB_TOKEN`: GitHub Personal Access Token (optional, but recommended to avoid rate limits).
- `SMTP_HOST`/`SMTP_PORT`/`SMTP_TIMEOUT`: Email server configuration.

### Running with Docker Compose

The easiest way to run the service along with its core dependencies (PostgreSQL and Mailpit):

1. Clone the repository:
   ```bash
   git clone https://github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur.git
   cd software-engineering-school-6-0-Oleg-amur
   ```

2. Copy `.env.example` to `.env` and adjust if necessary:
   ```bash
   cp .env.example .env
   ```

3. Start the services:
   ```bash
   docker compose up -d
   ```

The service will be available at:
- REST API: `http://localhost:8080`
- gRPC API: `localhost:50051`
- Release Tracker internal HTTP API: `http://localhost:8081`
- Mailpit UI (Email testing): `http://localhost:8025`
- Prometheus Metrics: `http://localhost:8080/metrics`

To start the full observability stack as well:

```bash
docker compose --profile observability up -d
```

Additional observability services will be available at:
- Prometheus UI: `http://localhost:9090`
- Grafana UI: `http://localhost:3000` (`admin` / `admin`)
- Elasticsearch: `http://localhost:9200`
- Kibana UI: `http://localhost:5601`

## Logging and Metrics

The application writes structured JSON logs to stdout. The `observability` Compose profile runs Filebeat, which tails Docker logs for the `subscription-service` Compose service, decodes the JSON payload, removes noisy Filebeat/Docker metadata, and sends events to Elasticsearch using Filebeat-managed storage.

The Kibana init container creates a `filebeat-*` data view and imports a `Subscription Service Logs` dashboard with recent structured log events filtered to `service.name: "subscription-service"`.

In Kibana, open logs in **Analytics -> Discover** and select the `Subscription Service Logs` data view. The imported log dashboard is under **Analytics -> Dashboard -> Subscription Service Logs**. If the data view is empty, generate at least one app request, for example `curl http://localhost:8080/health`, then wait a few seconds for Filebeat to publish the Docker log event.

Prometheus scrapes `subscription-service:8080/metrics` every 15 seconds. The application exposes RED metrics for both HTTP and gRPC traffic:
- `subscription_service_http_requests_total`
- `subscription_service_http_request_errors_total`
- `subscription_service_http_request_duration_seconds`
- `subscription_service_grpc_requests_total`
- `subscription_service_grpc_request_errors_total`
- `subscription_service_grpc_request_duration_seconds`

The same endpoint also exposes default Go runtime and process metrics, including CPU, RAM usage, heap usage, goroutines, and GC pauses. Grafana is provisioned automatically with Prometheus as the default datasource and a `Subscription Service Metrics` dashboard. The dashboard has an editable `rate_window` variable at the top for rate and percentile queries.

The Kibana init container is intentionally separate from Kibana itself so the dashboard/data-view import is repeatable and fails visibly when saved object import fails.

The `/health` endpoint checks database connectivity and returns `200 OK` when PostgreSQL is reachable or `503 Service Unavailable` when the database ping fails. Prometheus also exposes `subscription_service_database_up` and `subscription_service_database_ping_duration_seconds`; the dashboard shows the database up/down state.

## API Documentation

### REST API

The API documentation is available in Swagger format at `api/swagger.yaml`.

**Endpoints:**
- `POST /api/v1/subscribe`: Subscribe an email to a GitHub repository.
- `GET /api/v1/confirm/{token}`: Confirm the subscription.
- `GET /api/v1/unsubscribe/{token}`: Unsubscribe from notifications.
- `GET /api/v1/subscriptions?email=...`: List all subscriptions for an email.

Internal HTTP communication:

- `subscription-service -> release-tracker`: `POST /internal/v1/repositories/ensure` and `GET /internal/v1/repositories?repository=owner/repo`.
- REST baseline for `release-tracker -> subscription-service`: `GET /internal/v1/subscriptions?repository=owner/repo`.

Internal gRPC communication:

- `release-tracker -> subscription-service`: `subscriptions.v1.SubscriptionService/ListActiveSubscriptionsByRepository`.
- Change `useGRPCSubscriptionQueries` in `services/release-tracker/cmd/server/main.go` to switch the release tracker between the gRPC implementation and the preserved REST baseline.
- The shared contract is `shared/contracts/proto/subscriptions/v1/subscription_service.proto`.
- Run `buf lint` and `buf generate` (or `make proto-lint` and `make proto-generate`) after contract changes.

### gRPC API

The gRPC definition is available at `shared/contracts/proto/subscriptions/v1/subscription_service.proto`.

**Services:**
- `Subscribe`: Create a new subscription.
- `Confirm`: Confirm a subscription.
- `Unsubscribe`: Remove a subscription.
- `GetSubscriptions`: List all subscriptions for an email.
- `ListActiveSubscriptionsByRepository`: List active subscriptions for internal release processing.

## Project Structure

```text
├── shared/
│   └── contracts/             # Shared event contracts module
├── services/
│   ├── subscription-service/      # Subscription API and subscription database
│   ├── release-tracker/       # GitHub scanner and repository database
│   └── notification-worker/   # RabbitMQ consumer and SMTP delivery module
├── docs/                      # System design, C4 diagrams, and ADRs
├── test/e2e/                  # Whole-system Playwright tests
└── docker-compose.yaml        # Local runtime stack
```

## Release Detection Logic

The release tracker maintains a `last_seen_tag` for every tracked repository:
1. It fetches all active repositories from the database.
2. For each, it queries the GitHub API for the latest release.
3. If a new version is detected, it requests confirmed subscribers from `subscription-service` over HTTP and publishes notification jobs to RabbitMQ.
4. If rate limits are hit, the scanner gracefully skips the current cycle to wait for the window reset.

## Technical Considerations

- **Releases vs Tags**: Currently, the service monitors the GitHub "Releases" endpoint. Some repositories (like `golang/go`) primarily use git tags rather than official GitHub Releases. Future improvements could include fallback logic to monitor tags via Atom feeds or GraphQL if no releases are found.
- **Rate Limiting**: To avoid hitting GitHub's public API limits (60 req/hour), it is highly recommended to provide a `GITHUB_TOKEN`. This increases the limit to 5,000 requests per hour.

## Testing

To read more about tests and how to run them go to [testing.md](docs/testing.md)
