FROM golang:1.25.3-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download
COPY test/e2e/fakes/github ./test/e2e/fakes/github

RUN go build -o /app/fake-github ./test/e2e/fakes/github

FROM alpine:latest
COPY --from=builder /app/fake-github .

EXPOSE 8081
CMD ["./fake-github"]
