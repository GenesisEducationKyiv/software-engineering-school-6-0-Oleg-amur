# ADR 0002: Data Persistence

* **Status:** Accepted
* **Author:** Oleh Volkoboi
* **Date:** 2026-05-08

## Context
The application needs to store persistent data for subscribers, repositories being tracked, and the subscriptions linking them.

## Proposed Options

### Option 1: PostgreSQL
  Pros:
   - Excellent integration with Go.
   - Safely links data (subscribers to repositories) using strict foreign keys.
   - Highly reliable and prevents data duplication.

  Cons:
   - Requires hosting a separate database server.
   - Uses more server memory than SQLite.
   - Requires writing and managing schema migration files.

### Option 2: MySQL
  Pros:
   - Very common and easy to host.
   - Fast performance for simple read queries.

  Cons:
   - Requires hosting a separate database server.
   - Slightly weaker support for complex data types compared to PostgreSQL.
   - Requires writing and managing schema migration files.

### Option 3: MongoDB
  Pros:
   - No strict schema (easy to add new fields without migrations).
   - Easy to split across multiple servers as traffic grows.

  Cons:
   - Requires hosting a separate database server.
   - Cannot enforce strict links between tables (foreign keys). Linking subscribers to repositories must be handled manually in the Go code, increasing the risk of bugs.

### Option 4: SQLite
  Pros:
   - Zero setup required.
   - Stored as a single file directly next to the Go app.

  Cons:
   - Locks the whole database when saving data, making it slow for many concurrent users.
   - Cannot share the database file easily if we want to run multiple copies of the Go app.

## Decision
We chose **Option 1: PostgreSQL** for its robustness, strong support for relational integrity, and overall compatibility with the Go ecosystem.

### Schema
The database uses a normalized relational schema to maintain data integrity and avoid duplication.

![Database Schema](../diagrams/database-schema.drawio.png)

> [!TIP]
> The diagram above is an "Editable PNG". To modify it, download the file and export to [drawio](https://drawio.com/).

1.  **subscribers**: Stores unique user emails.
2.  **repositories**: Stores unique GitHub repositories (stored as `owner/repo` in the `name` column) and their last known release tag.
3.  **subscriptions**: Table with subscribtions that links subscribers to the repositories they are watching. It includes a unique `token` for managing the subscription (e.g., for unsubscription links) and a `subscription_status`.

### Data Retention
We decided on **No Immediate Deletion** for inactive records. Even if a subscription is removed, the associated `subscriber` and `repository` records are retained. This prioritizes efficiency for re-subscriptions and preserves historical context for future analytics. We can implement a periodic cleanup task later if database bloat becomes a concern.

## Consequences
- **Pros:**
    - Relational model fits the domain perfectly.
    - Ensures data integrity via foreign keys and constraints.
    - Facilitates clear and efficient queries.
- **Cons:**
    - Requires managing a database server.
    - Normalization necessitates JOIN operations.
