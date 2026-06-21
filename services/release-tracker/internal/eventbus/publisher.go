package eventbus

import (
	"context"
	"errors"
	"fmt"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	sharedrabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/messaging/rabbitmq"
	amqp "github.com/rabbitmq/amqp091-go"
)

type Publisher struct {
	connection *amqp.Connection
	channel    *amqp.Channel
	publisher  *sharedrabbitmq.JSONPublisher
}

func NewPublisher(cfg sharedrabbitmq.Config) (*Publisher, error) {
	connection, err := sharedrabbitmq.DialWithRetry(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("connect RabbitMQ: %w", err)
	}
	channel, err := connection.Channel()
	if err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}
	routingKeys := []string{
		events.SubscriptionConfirmationRequestedType,
		events.ReleaseNotificationRequestedType,
	}
	if err := sharedrabbitmq.DeclareTopology(channel, cfg, routingKeys); err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, err
	}
	publisher, err := sharedrabbitmq.NewJSONPublisher(channel, cfg.Exchange)
	if err != nil {
		_ = channel.Close()
		_ = connection.Close()
		return nil, err
	}
	return &Publisher{connection: connection, channel: channel, publisher: publisher}, nil
}

func (p *Publisher) Publish(ctx context.Context, event events.ReleaseNotificationRequested) error {
	return p.publisher.Publish(ctx, events.ReleaseNotificationRequestedType, event.EventID, event)
}

func (p *Publisher) Close() error {
	return errors.Join(p.channel.Close(), p.connection.Close())
}
