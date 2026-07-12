# HTTP vs gRPC Benchmarks

These benchmarks compare the preserved REST baseline and the new gRPC path for:

- HTTP: `GET /internal/v1/subscriptions?repository_id=...`
- gRPC: `subscriptions.v1.SubscriptionService/ListActiveSubscriptionsByRepository`

Both paths execute the same subscription-service use case and database query.

## Setup

Start the stack and seed benchmark data:

```sh
docker compose up --build -d subscription-service
make benchmark-seed
```

The seed creates 1000 active subscriptions for `repository_id=1`.

## autocannon

```sh
autocannon \
  --connections 50 \
  --amount 10000 \
  "http://127.0.0.1:8080/internal/v1/subscriptions?repository_id=1"
```

Empty-response baseline:

```sh
autocannon \
  --connections 50 \
  --amount 10000 \
  "http://127.0.0.1:8080/internal/v1/subscriptions?repository_id=999999"
```

## ghz

```sh
ghz \
  --insecure \
  --proto shared/contracts/proto/subscriptions/v1/subscription_service.proto \
  --call subscriptions.v1.SubscriptionService.ListActiveSubscriptionsByRepository \
  --data '{"repository_id": 1}' \
  --concurrency 50 \
  --connections 1 \
  --total 10000 \
  --format summary \
  127.0.0.1:50051
```

Empty-response baseline:

```sh
ghz \
  --insecure \
  --proto shared/contracts/proto/subscriptions/v1/subscription_service.proto \
  --call subscriptions.v1.SubscriptionService.ListActiveSubscriptionsByRepository \
  --data '{"repository_id": 999999}' \
  --concurrency 50 \
  --connections 1 \
  --total 10000 \
  --format summary \
  127.0.0.1:50051
```

## Go Client Benchmark

Use this for the most realistic service-to-service comparison:

```sh
go run ./benchmarks/subscription-client-compare \
  -mode all \
  -repository-id 1 \
  -total 10000 \
  -concurrency 50
```

Modes:

- `http-raw`: HTTP request + body read, without JSON decode.
- `http`: HTTP request + JSON decode + DTO mapping.
- `grpc`: gRPC request + protobuf decode + DTO mapping.

Why this benchmark is better for the final comparison:

- `autocannon` measures raw HTTP response reads and does not decode JSON.
- `ghz` uses a real gRPC client path and decodes protobuf.
- The Go benchmark mirrors the release-tracker adapters more closely on both sides.

Local run with `repository_id=1`, 1000 subscriptions, `total=10000`, `concurrency=50`:

| Mode | Throughput |
| --- | ---: |
| HTTP raw body read | 1155 req/s |
| HTTP + JSON decode | 826 req/s |
| gRPC + protobuf decode | 997 req/s |

For the production-like client path, gRPC was about `1.21x` faster than HTTP with JSON decode.
