# ADR 0006: Notification Worker and Modular Monolith

* **Status:** Accepted
* **Author:** Oleh Volkoboi
* **Date:** 2026-06-07

## Context
The application has two different kinds of responsibilities:

- user-facing API workflows and repository release tracking;
- email notification delivery.

These responsibilities change for different reasons. Subscription and release tracking logic is tightly connected to the main database, HTTP/gRPC APIs, and GitHub repository state. Email delivery is more operational: it deals with SMTP, templates, retries, acknowledgement, and delivery failures.

The original code grouped most release notifier responsibilities in broad packages such as `service`, `repository`, `scanner`, and `model`. That made package names simple, but it also mixed unrelated business areas and made module boundaries harder to see.

We need an architecture that:
- keeps the API and release tracking logic cohesive;
- isolates email delivery so SMTP failures and delivery retries do not complicate the core service;
- keeps the codebase understandable without turning every business capability into a separate microservice;
- makes future extraction possible only where there is a real operational reason.

## Proposed Options

### Option 1: Keep One Service With Layered Packages
- **Pros:** Simple deployment and minimal package count.
- **Cons:** Broad layers such as `service` and `repository` keep unrelated business capabilities together. Over time, subscription, release tracking, scanning, and notification code become harder to change independently.

### Option 2: Split Every Capability Into Separate Microservices
- **Pros:** Strong runtime isolation and independent deployment for each capability.
- **Cons:** Too much operational and coordination overhead for the current scope. Subscription management and release tracking share the same database model and are not independent enough to justify separate services.

### Option 3: Use a Modular Monolith for the Core Service and a Separate Notification Worker
- **Pros:** Keeps subscription and release tracking in one deployable service with explicit vertical module boundaries, while isolating email delivery in its own worker process.
- **Cons:** Requires clear package ownership rules to avoid modules importing each other casually.

### Option 4: Keep Notification Delivery Inside the Core Service
- **Pros:** Fewer processes and simpler local wiring.
- **Cons:** SMTP delivery failures, retries, and throughput concerns would live in the same process as user-facing APIs and scanning. This makes the core service less focused and harder to operate.

## Decision
We chose **Option 3: a modular monolith for the core service and a separate notification worker**.

The system is split into:
- `services/release-notifier`: the core service. It owns HTTP/gRPC APIs, subscription management, repository tracking, release scanning, and notification planning.
- `services/notification-worker`: the worker service. It owns email rendering and SMTP delivery.
- `shared/contracts`: shared event payloads used at the service boundary.

Inside `services/release-notifier`, we use vertical module slices instead of broad technical layers:
- `subscriptions`: subscription domain, persistence, usecases, and transport handlers.
- `releasetracker`: repository tracking domain, persistence, release scanning usecases, and scanner worker.
- `adapters`: external infrastructure implementations such as GitHub, PostgreSQL helpers, and notification publishing.

The `cmd/server` package is the composition root. It wires repositories, usecases, transports, workers, and infrastructure adapters together. Module packages define their own interfaces where they use dependencies.

Release notifications are planned in the core service, not in the notification worker. The worker receives ready-to-send notification jobs and does not read the subscription database. This keeps business decisions about who should receive a release notification close to subscription data, while keeping email delivery operationally isolated.

The specific event bus technology is intentionally outside this ADR. It will be documented separately.

## Consequences
- **Pros:**
    - The core service remains a single cohesive deployable unit for tightly related subscription and release tracking workflows.
    - Email delivery can be deployed, restarted, scaled, and observed independently from the API/scanner process.
    - Vertical module folders make ownership clearer than broad `service`, `repository`, and `model` packages.
    - Usecases can depend on small local interfaces instead of concrete infrastructure packages.
    - Future extraction remains possible without prematurely splitting every module into a microservice.
- **Cons:**
    - The core service still contains multiple business modules, so package boundaries need discipline.
    - Cross-module data needs careful handling to avoid one domain model becoming coupled to another module unnecessarily.
    - There is more wiring code in `cmd/server` than in a single layered package structure.
    - `shared/contracts` must remain small and limited to service-boundary payloads.

## Testing Strategy
- Unit tests validate each module usecase without external infrastructure.
- HTTP/gRPC integration tests wire the release notifier through its composition root and use test infrastructure to stay deterministic.
- End-to-end tests run the full deployed system, including the notification worker and Mailpit, to verify that notification delivery works across the service boundary.
