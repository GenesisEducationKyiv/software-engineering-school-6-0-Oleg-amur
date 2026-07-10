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

type SubscriptionSagaConsumer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	cfg  Config
	log  *slog.Logger
}

type SubscriptionSagaHandler interface {
	HandleSubscriptionConfirmationSucceeded(context.Context, events.SubscriptionConfirmationSucceeded) error
	HandleSubscriptionConfirmationFailed(context.Context, events.SubscriptionConfirmationFailed) error
}

func NewSubscriptionSagaConsumer(log *slog.Logger, cfg Config) (*SubscriptionSagaConsumer, error) {
	conn, err := sharedrabbitmq.DialWithRetry(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}

	if err := sharedrabbitmq.DeclareTopology(ch, cfg, subscriptionSagaRoutingKeys()); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	return &SubscriptionSagaConsumer{
		conn: conn,
		ch:   ch,
		cfg:  cfg,
		log:  log,
	}, nil
}

func (c *SubscriptionSagaConsumer) Subscribe(ctx context.Context, handler SubscriptionSagaHandler) error {
	consumer := sharedrabbitmq.NewJSONConsumer(c.log, c.ch, c.cfg.Queue)
	consumer.Handle(events.SubscriptionConfirmationSucceededType, func(ctx context.Context, body []byte) error {
		var event events.SubscriptionConfirmationSucceeded
		if err := json.Unmarshal(body, &event); err != nil {
			return fmt.Errorf("decode subscription confirmation succeeded event: %w", err)
		}
		if err := events.ValidateNotificationSchemaVersion(event.SchemaVersion); err != nil {
			return err
		}
		return handler.HandleSubscriptionConfirmationSucceeded(ctx, event)
	})
	consumer.Handle(events.SubscriptionConfirmationFailedType, func(ctx context.Context, body []byte) error {
		var event events.SubscriptionConfirmationFailed
		if err := json.Unmarshal(body, &event); err != nil {
			return fmt.Errorf("decode subscription confirmation failed event: %w", err)
		}
		if err := events.ValidateNotificationSchemaVersion(event.SchemaVersion); err != nil {
			return err
		}
		return handler.HandleSubscriptionConfirmationFailed(ctx, event)
	})
	return consumer.Subscribe(ctx)
}

func (c *SubscriptionSagaConsumer) Close() error {
	return errors.Join(c.ch.Close(), c.conn.Close())
}

func subscriptionSagaRoutingKeys() []string {
	return []string{
		events.SubscriptionConfirmationSucceededType,
		events.SubscriptionConfirmationFailedType,
	}
}
