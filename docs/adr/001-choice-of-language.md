# ADR 0001: Choice of Language

* **Status:** Accepted
* **Author:** Oleh Volkoboi
* **Date:** 2026-05-08

## Context
The project requires building a robust, high-performance service that handles HTTP and gRPC requests, interacts with external APIs (GitHub), and runs periodic background tasks.

## Proposed Options

### Option 1: Go (Golang)
- **Pros:** Personal expertise of developers, high performance, excellent concurrency primitives (goroutines), strong standard library, native gRPC support.
- **Cons:** Verbose error handling, unique type system.

### Option 2: Node.js (TypeScript)
- **Pros:** Fast development due to dynamic typing, huge ecosystem, familiar to many developers.
- **Cons:** Single-threaded (can struggle with CPU-bound tasks), gRPC support is not as "native" as in Go.

### Option 3: PHP
- **Pros:** Simple to set up for traditional web apps, wide hosting support.
- **Cons:** Not designed for long-running background processes (scanners), weaker gRPC ecosystem.

## Decision
We chose **Option 1: Go (Golang)** because it provides the best balance of performance, concurrency support for scanning tasks, and built-in tooling for both HTTP and gRPC. Also an important factor is that our product developers have experience with it.

## Consequences
- **Pros:**
    - High performance and efficiency.
    - Excellent support for concurrency for scanning and notification tasks.
    - Strong standard library, reducing external dependencies.
- **Cons:**
    - Error handling can be verbose.
    - Lack of syntax sugar that leads to boilerplate.
