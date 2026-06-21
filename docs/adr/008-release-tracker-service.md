# ADR 0008: Extract Release Tracker Service

* **Status:** Accepted
* **Author:** Oleh Volkoboi
* **Date:** 2026-06-21

## Context

Release tracking and subscription management previously ran in one process and shared one PostgreSQL schema. We need an existing synchronous HTTP boundary that can subsequently be migrated to gRPC and measured without changing the business operation under test.

## Decision

Extract release tracking into `services/release-tracker`.

- `release-notifier` owns subscribers, subscriptions, confirmation sagas, and their outbox.
- `release-tracker` owns tracked repositories, GitHub polling, and `last_seen_tag`.
- Each service has a separate PostgreSQL database and does not expose database IDs across the boundary.
- `release-notifier` calls release tracker over HTTP to ensure and read repository metadata.
- release tracker calls `release-notifier` over HTTP to list active subscriptions by repository name.
- release notification delivery remains asynchronous through RabbitMQ.

The internal HTTP contracts use the stable `owner/repo` repository name. The later gRPC implementation must preserve these use cases and leave the HTTP implementation available for comparison.

## Consequences

- Repository scanning can be deployed and scaled independently from the public subscription API.
- There are no cross-service foreign keys or shared tables.
- Subscription creation depends on release tracker availability.
- A scan that detects a release depends on the subscription query endpoint before publishing notification jobs.
- Listing subscriptions makes repository metadata calls to preserve the existing `last_seen_tag` response field.
