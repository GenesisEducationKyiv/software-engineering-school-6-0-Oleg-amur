FROM golang:1.25.3-alpine AS builder
ARG CGO_ENABLED=0
WORKDIR /workspace

COPY shared/contracts ./shared/contracts
COPY shared/messaging ./shared/messaging
COPY services/release-tracker ./services/release-tracker

WORKDIR /workspace/services/release-tracker
RUN go mod download
RUN go build -o /app/server ./cmd/server

FROM alpine:latest
COPY --from=builder /app/server .
COPY --from=builder /workspace/services/release-tracker/configs ./configs

EXPOSE 8081
CMD ["./server"]
