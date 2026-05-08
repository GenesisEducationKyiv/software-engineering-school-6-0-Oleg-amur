# Release Notifier Service

A Go-based service that monitors GitHub repositories for new releases and notifies subscribers via email. It provides both REST and gRPC interfaces for subscription management.

## Features

- **GitHub Monitoring**: Periodically scans GitHub repositories for new releases.
- **Email Notifications**: Notifies subscribers when a new release is detected.
- **Multi-Protocol Support**:
  - **REST API**: Standard HTTP endpoints for subscription management.
  - **gRPC API**: High-performance interface for service-to-service communication.
- **Persistence**: Uses PostgreSQL to store subscribers, repositories, and subscription states.
- **Monitoring**: Includes Prometheus metrics for service observability.
- **Dockerized**: Ready to run with Docker and Docker Compose.

## Tech Stack

- **Language**: [Go](https://go.dev/) (1.25+)
- **Database**: [PostgreSQL](https://www.postgresql.org/)
- **Communication**: [gRPC](https://grpc.io/), [net/http](https://pkg.go.dev/net/http)
- **Configuration**: [cleanenv](https://github.com/ilyakaznacheev/cleanenv)
- **Metrics**: [Prometheus](https://prometheus.io/)
- **Containerization**: [Docker](https://www.docker.com/)

## Getting Started

### Prerequisites

- [Docker](https://www.docker.com/get-started) and [Docker Compose](https://docs.docker.com/compose/install/)
- [Go](https://go.dev/doc/install) (optional, for local development)

### Configuration

The service is configured using environment variables or a YAML file. You can find an example configuration in `.env.example`.

Key configuration options:
- `DATABASE_URL`: PostgreSQL connection string.
- `SCAN_INTERVAL`: How often to check for new releases (e.g., `1m`, `1h`).
- `GITHUB_TOKEN`: GitHub Personal Access Token (optional, but recommended to avoid rate limits).
- `SMTP_HOST`/`SMTP_PORT`: Email server configuration.

### Running with Docker Compose

The easiest way to run the service along with its dependencies (PostgreSQL and Mailpit):

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
   docker-compose up -d
   ```

The service will be available at:
- REST API: `http://localhost:8080`
- gRPC API: `localhost:50051`
- Mailpit UI (Email testing): `http://localhost:8025`
- Prometheus Metrics: `http://localhost:8080/metrics`

## API Documentation

### REST API

The API documentation is available in Swagger format at `api/swagger.yaml`.

**Endpoints:**
- `POST /api/v1/subscribe`: Subscribe an email to a GitHub repository.
- `GET /api/v1/confirm/{token}`: Confirm the subscription.
- `GET /api/v1/unsubscribe/{token}`: Unsubscribe from notifications.
- `GET /api/v1/subscriptions?email=...`: List all subscriptions for an email.

### gRPC API

The gRPC definition is available at `api/proto/release_notifier.proto`.

**Services:**
- `Subscribe`: Create a new subscription.
- `Confirm`: Confirm a subscription.
- `Unsubscribe`: Remove a subscription.
- `GetSubscriptions`: List all subscriptions for an email.

## Project Structure

```text
├── api/               # API definitions (Swagger/Proto)
├── cmd/               # Service entry points
├── configs/           # Configuration files
├── internal/          # Private application code
│   ├── api/           # Transport layers (HTTP/gRPC)
│   ├── apperr/        # Domain errors
│   ├── config/        # Configuration loading
│   ├── database/      # DB initialization and migrations
│   ├── github/        # GitHub API client
│   ├── models/        # Domain entities
│   ├── notifier/      # Email notification logic
│   ├── repository/    # Database persistence
│   ├── scanner/       # Release monitoring logic
│   └── service/       # Business logic layer
├── migrations/        # SQL migration files
└── docs/              # Additional documentation
```

## Architecture & Design Decisions

- **Web Framework**: Built using standard `net/http` for the REST API to keep dependencies "thin" and leverage Go's powerful standard library.
- **Database Schema**: Uses a normalized structure with separate tables for `Subscribers`, `Repositories`, and `Subscriptions`. This allows for efficient data management and prevents duplication (e.g., one repository being scanned once even if it has multiple subscribers).
- **GitHub Client**: Custom implementation of the GitHub API client to handle rate limiting (429 Too Many Requests) and provide specific functionality needed for release monitoring without the overhead of larger libraries.
- **Background Scanner**: Uses a Go-native `time.Timer` for periodic scanning, avoiding external cron dependencies and ensuring a self-contained monolith.

## Release Detection Logic

The service maintains a `last_seen_tag` for every tracked repository:
1. It fetches all active repositories from the database.
2. For each, it queries the GitHub API for the latest release.
3. If a new version is detected (different from `last_seen_tag`), it triggers email notifications to all confirmed subscribers of that repository.
4. If rate limits are hit, the scanner gracefully skips the current cycle to wait for the window reset.

## Technical Considerations

- **Releases vs. Tags**: Currently, the service monitors the GitHub "Releases" endpoint. Some repositories (like `golang/go`) primarily use git tags rather than official GitHub Releases. Future improvements could include fallback logic to monitor tags via Atom feeds or GraphQL if no releases are found.
- **Rate Limiting**: To avoid hitting GitHub's public API limits (60 req/hour), it is highly recommended to provide a `GITHUB_TOKEN`. This increases the limit to 5,000 requests per hour.

## Testing

Run unit tests:
```bash
go test ./...
```
