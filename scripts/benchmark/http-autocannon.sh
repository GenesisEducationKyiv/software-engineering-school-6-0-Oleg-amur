#!/usr/bin/env sh
set -eu

if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

if ! command -v autocannon >/dev/null 2>&1; then
    echo "autocannon is not installed. Install it with: npm install -g autocannon" >&2
    exit 127
fi

: "${SUBSCRIPTION_SERVICE_HTTP_PORT:=8080}"
: "${REPOSITORY_ID:=1}"
: "${DURATION_SECONDS:=30}"
: "${CONCURRENCY:=50}"
: "${RESULTS_DIR:=benchmarks/results}"
: "${HTTP_URL:=http://localhost:${SUBSCRIPTION_SERVICE_HTTP_PORT}/internal/v1/subscriptions?repository_id=${REPOSITORY_ID}}"

mkdir -p "$RESULTS_DIR"
timestamp="$(date -u +%Y%m%dT%H%M%SZ)"
output="$RESULTS_DIR/http-autocannon-${timestamp}.json"

autocannon \
    --connections "$CONCURRENCY" \
    --duration "$DURATION_SECONDS" \
    --json \
    "$HTTP_URL" | tee "$output"

echo "Saved HTTP benchmark report to $output" >&2
