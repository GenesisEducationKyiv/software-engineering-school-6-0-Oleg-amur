# ADR 0006: Notification Worker and Event Bus

* **Status:** Accepted
* **Author:** Oleh Volkoboi
* **Date:** 2026-06-07

## Context
The previous implementation delivered subscription confirmation and release notification events through in-process Go channels.
That made notification delivery simple, but messages could be lost when the process stopped or when a channel buffer filled.

The application also needs clearer module boundaries:
- subscription and release tracking own the business data;
- notification delivery owns email rendering, SMTP delivery, acknowledgement, and failure handling;
- the message broker must be replaceable, so application services must not depend on RabbitMQ-specific APIs.

## Proposed Options

### Option 1: Keep In-Process Channels
- **Pros:** Simple, fast, easy to unit test.
- **Cons:** No durable delivery, no separate worker, no inter-service communication boundary.

### Option 2: Add Transactional Outbox
- **Pros:** Strongest consistency between database writes and event publication.
- **Cons:** More schema, polling, deduplication, and operational complexity than required for the current project scope.

### Option 3: Use RabbitMQ Behind an Event Bus Port
- **Pros:** Durable queue, manual acknowledgements, dead-letter queue, separate notification worker, and a clean adapter boundary.
- **Cons:** Does not provide full atomicity between PostgreSQL commits and event publication without an outbox.

### Option 4: Use Kafka Behind the Same Event Bus Port
- **Pros:** Durable event log, replay, retention, independent consumer groups.
- **Cons:** Heavier operational model and unnecessary event-log semantics for notification jobs.

## Decision
We chose **Option 3: RabbitMQ behind an event bus port**.

The core application depends on `NotificationPublisher`, not on RabbitMQ. RabbitMQ is implemented as an infrastructure adapter.
A future Kafka adapter can implement the same application-facing port without changing subscription, scanner, or notification delivery logic.

The notification module is split into:
- `services/release-notifier`: HTTP/gRPC API, subscription management, repository scanning, and notification planning.
- `services/notification-worker`: RabbitMQ consumer and SMTP email delivery.
- `shared/contracts`: shared event payloads and event type constants used by both services.

Release notifications are planned in the core service, not in the worker. The worker receives one ready-to-send notification job per recipient and does not read the subscription database.

## Consequences
- **Pros:**
    - Notification delivery survives application process restarts after RabbitMQ has confirmed the publish.
    - The worker acknowledges messages only after successful SMTP delivery.
    - Failed messages are dead-lettered instead of silently lost.
    - RabbitMQ is isolated behind an adapter and can be replaced by Kafka later.
    - HTTP/gRPC integration tests can use an in-memory event publisher while e2e tests cover the RabbitMQ-backed worker.
- **Cons:**
    - Full atomicity between database writes and event publication is not guaranteed without an outbox.
    - If the API process crashes after writing to PostgreSQL but before publishing to RabbitMQ, a confirmation event may be missed.
    - If release notification jobs are published but the repository tag update fails, the next scan may publish duplicate jobs.

## Testing Strategy
- Unit tests validate application services and the notification worker handler without RabbitMQ.
- HTTP/gRPC integration tests use an in-memory publisher to keep them deterministic and focused on API behavior.
- End-to-end tests run RabbitMQ and `notification-worker` in Docker Compose to verify real async delivery to Mailpit.
