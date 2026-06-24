FROM golang:1.25.3-alpine AS builder
ARG CGO_ENABLED=0
WORKDIR /workspace

COPY shared/contracts ./shared/contracts
COPY shared/messaging ./shared/messaging
COPY services/subscription-service ./services/subscription-service

WORKDIR /workspace/services/subscription-service
RUN go mod download
RUN go build -o /app/server ./cmd/server

FROM alpine:latest
COPY --from=builder /app/server .
COPY --from=builder /workspace/services/subscription-service/configs ./configs

EXPOSE 8080
CMD ["./server"]
