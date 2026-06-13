package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	sharedrabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/messaging/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Config = sharedrabbitmq.Config

type Consumer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	cfg  Config
	log  *slog.Logger
}

type NotificationHandler interface {
	HandleSubscriptionConfirmationRequested(context.Context, events.SubscriptionConfirmationRequested) error
	HandleReleaseNotificationRequested(context.Context, events.ReleaseNotificationRequested) error
}

func NewNotificationConsumer(log *slog.Logger, cfg Config) (*Consumer, error) {
	conn, err := sharedrabbitmq.DialWithRetry(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}

	if err := sharedrabbitmq.DeclareTopology(ch, cfg, notificationRoutingKeys()); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	return &Consumer{
		conn: conn,
		ch:   ch,
		cfg:  cfg,
		log:  log,
	}, nil
}

func (c *Consumer) Subscribe(ctx context.Context, handler NotificationHandler) error {
	return c.notificationConsumer(handler).Subscribe(ctx)
}

func (c *Consumer) handleDelivery(
	ctx context.Context,
	handler NotificationHandler,
	delivery amqp.Delivery,
) {
	c.notificationConsumer(handler).HandleDelivery(ctx, delivery)
}

func (c *Consumer) notificationConsumer(handler NotificationHandler) *sharedrabbitmq.JSONConsumer {
	consumer := sharedrabbitmq.NewJSONConsumer(c.log, c.ch, c.cfg.Queue)
	consumer.Handle(events.SubscriptionConfirmationRequestedType, func(ctx context.Context, body []byte) error {
		var event events.SubscriptionConfirmationRequested
		if err := json.Unmarshal(body, &event); err != nil {
			return fmt.Errorf("decode subscription confirmation event: %w", err)
		}
		return handler.HandleSubscriptionConfirmationRequested(ctx, event)
	})
	consumer.Handle(events.ReleaseNotificationRequestedType, func(ctx context.Context, body []byte) error {
		var event events.ReleaseNotificationRequested
		if err := json.Unmarshal(body, &event); err != nil {
			return fmt.Errorf("decode release notification event: %w", err)
		}
		return handler.HandleReleaseNotificationRequested(ctx, event)
	})
	return consumer
}

func (c *Consumer) Close() error {
	return errors.Join(c.ch.Close(), c.conn.Close())
}

func notificationRoutingKeys() []string {
	return []string{
		events.SubscriptionConfirmationRequestedType,
		events.ReleaseNotificationRequestedType,
	}
}
