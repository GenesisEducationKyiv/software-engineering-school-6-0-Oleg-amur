//go:build integration

package eventbus_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/adapters/eventbus/rabbitmq"
	"github.com/google/uuid"
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/stretchr/testify/require"
)

func notificationConfig(url string) rabbitmq.Config {
	suffix := uuid.NewString()

	return rabbitmq.Config{
		URL:      url,
		Exchange: "test.notifications." + suffix,
		Queue:    "test.notification-worker." + suffix,
		DLQ:      "test.notification-worker.dlq." + suffix,
	}
}

func openRabbitMQChannel(t *testing.T, url string) (*amqp.Connection, *amqp.Channel) {
	t.Helper()

	conn, err := amqp.DialConfig(url, amqp.Config{
		Dial: amqp.DefaultDial(5 * time.Second),
	})
	require.NoError(t, err)

	ch, err := conn.Channel()
	if err != nil {
		require.NoError(t, conn.Close())
		t.Fatalf("open RabbitMQ channel: %v", err)
	}

	return conn, ch
}

func requireDelivery(ctx context.Context, t *testing.T, ch *amqp.Channel, queue string) amqp.Delivery {
	t.Helper()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		delivery, ok, err := ch.Get(queue, true)
		require.NoError(t, err)
		if ok {
			return delivery
		}

		select {
		case <-ctx.Done():
			t.Fatalf("receive RabbitMQ delivery from %q: %v", queue, ctx.Err())
		case <-ticker.C:
		}
	}
}

func requireMessageProperties(t *testing.T, delivery amqp.Delivery, messageType, messageID string) {
	t.Helper()

	require.Equal(t, messageType, delivery.Type)
	require.Equal(t, messageType, delivery.RoutingKey)
	require.Equal(t, messageID, delivery.MessageId)
	require.Equal(t, "application/json", delivery.ContentType)
	require.Equal(t, uint8(amqp.Persistent), delivery.DeliveryMode)
	require.False(t, delivery.Timestamp.IsZero())
}

func requireJSONPayload[T comparable](t *testing.T, body []byte, want T) {
	t.Helper()

	var got T
	require.NoError(t, json.Unmarshal(body, &got))
	require.Equal(t, want, got)
}

func closeRabbitMQ(t *testing.T, conn *amqp.Connection, ch *amqp.Channel) {
	t.Helper()

	require.NoError(t, ch.Close())
	require.NoError(t, conn.Close())
}
