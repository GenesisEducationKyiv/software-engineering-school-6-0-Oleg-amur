FROM golang:1.25.3-alpine AS builder
WORKDIR /app

COPY test/e2e/fakes/github/go.mod ./
RUN go mod download
COPY test/e2e/fakes/github ./

RUN go build -o /app/fake-github .

FROM alpine:latest
COPY --from=builder /app/fake-github .

EXPOSE 8081
CMD ["./fake-github"]
