#!/usr/bin/env sh
set -eu

if [ -f .env ]; then
    set -a
    . ./.env
    set +a
fi

: "${SUBSCRIPTION_SERVICE_DB_USER:=postgres}"
: "${SUBSCRIPTION_SERVICE_DB_NAME:=subscription_service}"
: "${REPOSITORY_ID:=1}"
: "${SUBSCRIPTION_COUNT:=1000}"

docker compose exec -T subscription-service-postgresql \
    psql \
    -U "$SUBSCRIPTION_SERVICE_DB_USER" \
    -d "$SUBSCRIPTION_SERVICE_DB_NAME" \
    -v "repository_id=$REPOSITORY_ID" \
    -v "subscription_count=$SUBSCRIPTION_COUNT" \
    -f /dev/stdin < benchmarks/seed-subscription-data.sql
