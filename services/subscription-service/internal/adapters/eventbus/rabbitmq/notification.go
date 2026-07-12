package rabbitmq

import (
	"context"
	"errors"
	"fmt"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	sharedrabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/messaging/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Config = sharedrabbitmq.Config

type Publisher struct {
	conn      *amqp.Connection
	ch        *amqp.Channel
	publisher *sharedrabbitmq.JSONPublisher
}

func NewNotificationPublisher(cfg Config) (*Publisher, error) {
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

	publisher, err := sharedrabbitmq.NewJSONPublisher(ch, cfg.Exchange)
	if err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	return &Publisher{
		conn:      conn,
		ch:        ch,
		publisher: publisher,
	}, nil
}

func (p *Publisher) PublishSubscriptionConfirmation(
	ctx context.Context,
	event events.SubscriptionConfirmationRequested,
) error {
	return p.publish(ctx, events.SubscriptionConfirmationRequestedType, event.EventID, event)
}

func (p *Publisher) PublishReleaseNotification(
	ctx context.Context,
	event events.ReleaseNotificationRequested,
) error {
	return p.publish(ctx, events.ReleaseNotificationRequestedType, event.EventID, event)
}

func (p *Publisher) publish(ctx context.Context, routingKey, eventID string, event any) error {
	return p.publisher.Publish(ctx, routingKey, eventID, event)
}

func (p *Publisher) Close() error {
	return errors.Join(p.ch.Close(), p.conn.Close())
}

func notificationRoutingKeys() []string {
	return []string{
		events.SubscriptionConfirmationRequestedType,
		events.ReleaseNotificationRequestedType,
	}
}
