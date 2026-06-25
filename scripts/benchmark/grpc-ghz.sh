#!/usr/bin/env sh
set -eu

if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

if ! command -v ghz >/dev/null 2>&1; then
    echo "ghz is not installed. Install it with: go install github.com/bojand/ghz/cmd/ghz@latest" >&2
    exit 127
fi

: "${SUBSCRIPTION_SERVICE_GRPC_PORT:=50051}"
: "${REPOSITORY_ID:=1}"
: "${DURATION_SECONDS:=30}"
: "${CONCURRENCY:=50}"
: "${GRPC_CONNECTIONS:=50}"
: "${RESULTS_DIR:=benchmarks/results}"
: "${GRPC_ADDRESS:=localhost:${SUBSCRIPTION_SERVICE_GRPC_PORT}}"

mkdir -p "$RESULTS_DIR"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
output="$RESULTS_DIR/grpc-ghz-${timestamp}.json"

ghz \
    --insecure \
    --proto shared/contracts/proto/subscriptions/v1/subscription_service.proto \
    --call subscriptions.v1.SubscriptionService.ListActiveSubscriptionsByRepository \
    --data "{\"repository_id\": ${REPOSITORY_ID}}" \
    --concurrency "$CONCURRENCY" \
    --connections "$GRPC_CONNECTIONS" \
    --duration "${DURATION_SECONDS}s" \
    --duration-stop wait \
    --format json \
    --output "$output" \
    "$GRPC_ADDRESS"

echo "Saved gRPC benchmark report to $output" >&2
