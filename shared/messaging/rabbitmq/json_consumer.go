package rabbitmq

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	amqp "github.com/rabbitmq/amqp091-go"
)

type JSONHandler func(ctx context.Context, body []byte) error

type JSONConsumer struct {
	ch       *amqp.Channel
	queue    string
	log      *slog.Logger
	handlers map[string]JSONHandler
}

func NewJSONConsumer(log *slog.Logger, ch *amqp.Channel, queue string) *JSONConsumer {
	return &JSONConsumer{
		ch:       ch,
		queue:    queue,
		log:      log,
		handlers: make(map[string]JSONHandler),
	}
}

func (c *JSONConsumer) Handle(routingKey string, handler JSONHandler) {
	c.handlers[routingKey] = handler
}

func (c *JSONConsumer) Subscribe(ctx context.Context) error {
	deliveries, err := c.ch.Consume(
		c.queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume queue: %w", err)
	}

	c.log.Info("consumer started", "queue", c.queue)

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("RabbitMQ deliveries channel closed")
			}
			c.HandleDelivery(ctx, delivery)
		}
	}
}

func (c *JSONConsumer) HandleDelivery(ctx context.Context, delivery amqp.Delivery) {
	if err := c.dispatch(ctx, delivery); err != nil {
		c.log.Error(
			"message handling failed",
			"type",
			delivery.RoutingKey,
			"message_id",
			delivery.MessageId,
			"error",
			err,
		)
		if nackErr := delivery.Nack(false, false); nackErr != nil {
			c.log.Error("failed to nack message", "message_id", delivery.MessageId, "error", nackErr)
		}
		return
	}

	if err := delivery.Ack(false); err != nil {
		c.log.Error("failed to ack message", "message_id", delivery.MessageId, "error", err)
	}
}

func (c *JSONConsumer) dispatch(ctx context.Context, delivery amqp.Delivery) error {
	handler, ok := c.handlers[delivery.RoutingKey]
	if !ok {
		return fmt.Errorf("unknown message type: %s", delivery.RoutingKey)
	}

	return handler(ctx, delivery.Body)
}
