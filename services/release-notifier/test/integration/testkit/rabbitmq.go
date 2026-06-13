//go:build integration

package testkit

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

const rabbitMQImage = "rabbitmq:4.2-management-alpine"

type RabbitMQ struct {
	URL       string
	Container testcontainers.Container
}

func NewRabbitMQ(ctx context.Context, t testing.TB) *RabbitMQ {
	t.Helper()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:        rabbitMQImage,
			ExposedPorts: []string{"5672/tcp"},
			WaitingFor: wait.ForAll(
				wait.ForListeningPort("5672/tcp"),
				wait.ForLog("Server startup complete"),
			).WithStartupTimeout(2 * time.Minute),
		},
		Started: true,
	})
	require.NoError(t, err, "start rabbitmq container")

	host, err := container.Host(ctx)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		t.Fatalf("get rabbitmq host: %v", err)
	}

	port, err := container.MappedPort(ctx, "5672/tcp")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		t.Fatalf("get rabbitmq port: %v", err)
	}

	rabbit := &RabbitMQ{
		URL:       fmt.Sprintf("amqp://guest:guest@%s:%s/", host, port.Port()),
		Container: container,
	}
	t.Cleanup(func() {
		require.NoError(t, rabbit.Close(), "close rabbitmq")
	})

	return rabbit
}

func (r *RabbitMQ) Close() error {
	var errs []error
	if r.Container != nil {
		if err := testcontainers.TerminateContainer(r.Container); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
