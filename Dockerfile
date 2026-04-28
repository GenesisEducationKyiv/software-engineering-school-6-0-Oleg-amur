FROM golang:1.25.3-alpine as builder
ARG CGO_ENABLED=0
WORKDIR /app

COPY go.mod ./
RUN go mod download
COPY . .

RUN go build -o /app/server ./cmd/server/main.go

FROM alpine:latest 
COPY --from=builder /app/server .
COPY --from=builder /app/configs ./configs

EXPOSE 8080
CMD ["./server"]
