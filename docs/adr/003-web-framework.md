# ADR 0003: Web Framework & Communication

* **Status:** Accepted
* **Author:** Oleh Volkoboi
* **Date:** 2026-05-08

## Context
The service needs to provide a RESTful API for standard web interactions and a more efficient interface for internal or high-performance communication.

## Proposed Options

### Option 1: Standard Library (`net/http`) + gRPC
- **Pros:** Minimal dependencies for REST, high-performance binary serialization with gRPC, strong typing, better for learning purposes.
- **Cons:** More boilerplate for REST routing compared to frameworks.

### Option 2: High-level Frameworks (e.g., Gin, Echo)
- **Pros:** Faster initial development, less boilerplate code.
- **Cons:** Introduces external dependencies, adds "magic" that can obscure core logic.

## Decision
We chose **Option 1: Standard Library (`net/http`) + gRPC** to keep the core lightweight while providing both a standard RESTful interface and a high-performance, type-safe gRPC alternative.

## Consequences
- **Pros:**
    - Zero external dependencies for the HTTP layer.
    - Excellent performance and type safety for gRPC users.
    - Better for leaning purposes
- **Cons:**
    - Requires maintaining both Swagger (REST) and Protobuf (gRPC) definitions.
    - More manual work for HTTP parameter validation.
