# ADR 0004: Background Scanning

* **Status:** Accepted
* **Author:** Oleh Volkoboi
* **Date:** 2026-05-08

## Context
The service must regularly check for new releases in tracked GitHub repositories and notify subscribers.

## Proposed Options

### Option 1: Timer-based Internal Scanner
An internal loop using Go's `time.Ticker` for periodic scanning.
- **Pros:** Simple implementation, self-contained within the Go binary, no external infrastructure dependencies.
- **Cons:** Horizontal scaling requires distributed locks (e.g., Postgres advisory locks) to prevent duplicate scans.

### Option 2: Cron Jobs
Using the system's cron daemon to trigger periodic scanning operation.
- **Pros:** Standardized OS-level scheduling.
- **Cons:** Harder to manage within a containerized environment; splits application logic between the binary and the OS.

## Decision
We chose **Option 1: Timer-based Internal Scanner** for its simplicity and alignment with the monolith architecture. It keeps the service self-contained and easy to deploy. 

Regardless of the scanning mechanism, **SMTP** is used as the common transport layer for notifications, integrated with **Mailpit** for local development.

## Consequences
- **Pros:**
    - Easy to test locally using Docker Compose.
    - No external dependencies.
- **Cons:**
    - Potential for duplicate scans if multiple instances are started without a locking mechanism.
    - Less flexible than a dedicated job queue for complex retry logic.
