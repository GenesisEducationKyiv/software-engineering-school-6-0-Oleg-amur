# ADR 0009: Internal Subscription Query over gRPC

* **Status:** Accepted
* **Author:** Oleh Volkoboi
* **Date:** 2026-06-24

## Context

The release tracker synchronously requests active subscriptions from the subscription service when it detects a new release. The existing REST endpoint provides a baseline that must remain available for a protocol comparison.

## Decision

Add a unary gRPC operation, `SubscriptionService.ListActiveSubscriptionsByRepository`, alongside the existing REST endpoint.

- All subscription RPCs use one protobuf contract in `shared/contracts/proto/subscriptions/v1` because multiple services consume it.
- Buf owns protobuf linting and code generation.
- The subscription service exposes REST and gRPC implementations concurrently.
- The release tracker keeps both client adapters behind its existing `Subscriptions` interface.
- The composition root selects the active adapter through `useGRPCSubscriptionQueries`; changing one constant switches between the implementations without changing domain code.
- Internal traffic uses plaintext gRPC inside the Compose network, matching the existing plaintext HTTP baseline.

## Consequences

- HTTP and gRPC execute the same use case and database query, making throughput comparisons meaningful.
- Generated contracts are shared without importing another service's `internal` packages.
- The transport choice is deliberately compile-time for this homework; runtime configuration can be introduced later if operational switching becomes necessary.
- Both implementations must remain covered and compatible while the comparison is maintained.
