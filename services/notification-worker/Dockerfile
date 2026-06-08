FROM golang:1.25.3-alpine AS builder
ARG CGO_ENABLED=0
WORKDIR /workspace

COPY shared/contracts ./shared/contracts
COPY services/notification-worker ./services/notification-worker

WORKDIR /workspace/services/notification-worker
RUN GOWORK=off go mod download
RUN GOWORK=off go build -o /app/notification-worker ./cmd/notification-worker

FROM alpine:latest
COPY --from=builder /app/notification-worker .
COPY --from=builder /workspace/services/notification-worker/configs ./configs

CMD ["./notification-worker"]
