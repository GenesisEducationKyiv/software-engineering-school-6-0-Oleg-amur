# ADR 0007: Message Broker

* **Status:** Accepted
* **Author:** Oleh Volkoboi
* **Date:** 2026-06-13

## Context
The release notifier now has a separate notification worker. The core service creates notification commands when:

- a user subscribes and needs a confirmation email;
- a new release is detected and confirmed subscribers need release emails.

The notification worker consumes those commands and sends emails through SMTP.

This communication should not be in-process. Email delivery can be slow or temporarily unavailable, and the API/scanner process should not block on SMTP delivery. Notification jobs also need acknowledgement semantics: if the worker fails to process a message, the message should not be silently lost.

We need a broker that supports:
- asynchronous communication between `subscription-service` and `notification-worker`;
- durable notification jobs;
- manual acknowledgement after SMTP delivery;
- dead-lettering failed jobs for inspection;
- simple routing for different notification command types.

## Proposed Options

### Option 1: Redis Pub/Sub
- **Pros:** Simple to start locally, low latency, familiar operational model.
- **Cons:** Redis Pub/Sub is ephemeral. If the notification worker is offline, published messages can be missed. It also does not provide the acknowledgement and dead-letter semantics required for reliable notification jobs.

### Option 2: Redis Streams
- **Pros:** Persistent stream, consumer groups, acknowledgement support.
- **Cons:** More suitable than Redis Pub/Sub, but still less direct for this project than a queue broker. Dead-letter handling and routing conventions need more application-level design.

### Option 3: Kafka
- **Pros:** Durable event log, replay, retention, high throughput, consumer groups.
- **Cons:** Operationally heavier than needed. The system needs a work queue for notification commands, not a long-lived analytical event log. Kafka would add concepts such as partitions, offsets, retention, and replay that are not currently required.

### Option 4: RabbitMQ
- **Pros:** Durable queues, direct exchanges, routing keys, manual acknowledgements, publisher confirms, and dead-letter queues map directly to notification job delivery.
- **Cons:** Adds broker infrastructure and AMQP-specific operational knowledge.

## Decision
We chose **Option 4: RabbitMQ**.

The system publishes notification commands to RabbitMQ from `subscription-service`, and `notification-worker` consumes them. The current command types are:

- `notification.subscription_confirmation_requested`
- `notification.release_notification_requested`

RabbitMQ is a good fit because notification delivery is command/job oriented:

- each message should be processed by a worker;
- the worker should acknowledge only after successful SMTP delivery;
- failed messages should be rejected without requeue and sent to a dead-letter queue;
- the broker should keep jobs while the worker is temporarily unavailable.

RabbitMQ is kept behind adapter packages. Application usecases depend on small publisher/handler interfaces and shared event contracts, not on AMQP types.

## Consequences
- **Pros:**
    - Notification jobs survive worker restarts after they are accepted by the broker.
    - Manual acknowledgements protect against losing messages during worker failures.
    - Dead-letter queues make failed notification jobs visible for debugging or later replay.
    - Direct exchange routing is enough for the current command types.
    - The implementation remains simpler than Kafka for this workload.
- **Cons:**
    - Full atomicity between PostgreSQL writes and RabbitMQ publishes is not guaranteed without an outbox.
    - Duplicate notification jobs are still possible if publishing succeeds but a later database update fails.
    - Local and production environments must run RabbitMQ.
    - AMQP-specific topology needs to be maintained by the adapters.

## Testing Strategy
- Unit tests cover publisher-facing usecases with in-memory publishers.
- Unit tests cover notification worker handling without RabbitMQ by mocking email sender and message builder.
- RabbitMQ consumer adapter tests cover dispatch, acknowledgement, negative acknowledgement, malformed JSON, unknown routing keys, and handler failures without starting a broker.
- End-to-end tests cover the full Docker Compose flow with RabbitMQ, notification worker, and Mailpit.
