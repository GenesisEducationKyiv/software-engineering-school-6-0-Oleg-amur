package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/contracts/events"
	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	contentTypeJSON = "application/json"
	publishTimeout  = 5 * time.Second
	dialTimeout     = 30 * time.Second
	dialInterval    = 500 * time.Millisecond
)

type Config struct {
	URL      string
	Exchange string
	Queue    string
	DLQ      string
}

type Publisher struct {
	conn      *amqp.Connection
	ch        *amqp.Channel
	exchange  string
	confirms  <-chan amqp.Confirmation
	publishMu sync.Mutex
}

func NewNotificationPublisher(cfg Config) (*Publisher, error) {
	conn, err := dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}

	if err := declareTopology(ch, cfg); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, err
	}

	if err := ch.Confirm(false); err != nil {
		_ = ch.Close()
		_ = conn.Close()
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	return &Publisher{
		conn:     conn,
		ch:       ch,
		exchange: cfg.Exchange,
		confirms: ch.NotifyPublish(make(chan amqp.Confirmation, 1)),
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
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	p.publishMu.Lock()
	defer p.publishMu.Unlock()

	publishCtx, cancel := context.WithTimeout(ctx, publishTimeout)
	defer cancel()

	if err := p.ch.PublishWithContext(
		publishCtx,
		p.exchange,
		routingKey,
		false,
		false,
		amqp.Publishing{
			ContentType:  contentTypeJSON,
			DeliveryMode: amqp.Persistent,
			Type:         routingKey,
			MessageId:    eventID,
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	); err != nil {
		return fmt.Errorf("publish event: %w", err)
	}

	select {
	case confirmation := <-p.confirms:
		if !confirmation.Ack {
			return errors.New("RabbitMQ publish was not acknowledged")
		}
		return nil
	case <-publishCtx.Done():
		return publishCtx.Err()
	}
}

func (p *Publisher) Close() error {
	return errors.Join(p.ch.Close(), p.conn.Close())
}

type Consumer struct {
	conn *amqp.Connection
	ch   *amqp.Channel
	cfg  Config
	log  *slog.Logger
}

func NewNotificationConsumer(log *slog.Logger, cfg Config) (*Consumer, error) {
	conn, err := dial(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("connect to RabbitMQ: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("open RabbitMQ channel: %w", err)
	}

	if err := declareTopology(ch, cfg); err != nil {
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

func (c *Consumer) Subscribe(ctx context.Context, handler interface {
	HandleSubscriptionConfirmationRequested(context.Context, events.SubscriptionConfirmationRequested) error
	HandleReleaseNotificationRequested(context.Context, events.ReleaseNotificationRequested) error
},
) error {
	deliveries, err := c.ch.Consume(
		c.cfg.Queue,
		"",
		false,
		false,
		false,
		false,
		nil,
	)
	if err != nil {
		return fmt.Errorf("consume notification queue: %w", err)
	}

	c.log.Info("notification consumer started", "queue", c.cfg.Queue)

	for {
		select {
		case <-ctx.Done():
			return nil
		case delivery, ok := <-deliveries:
			if !ok {
				return errors.New("RabbitMQ deliveries channel closed")
			}
			c.handleDelivery(ctx, handler, delivery)
		}
	}
}

func (c *Consumer) handleDelivery(ctx context.Context, handler interface {
	HandleSubscriptionConfirmationRequested(context.Context, events.SubscriptionConfirmationRequested) error
	HandleReleaseNotificationRequested(context.Context, events.ReleaseNotificationRequested) error
}, delivery amqp.Delivery,
) {
	if err := c.dispatch(ctx, handler, delivery); err != nil {
		c.log.Error(
			"notification event failed",
			"type",
			delivery.RoutingKey,
			"message_id",
			delivery.MessageId,
			"error",
			err,
		)
		if nackErr := delivery.Nack(false, false); nackErr != nil {
			c.log.Error("failed to nack notification event", "message_id", delivery.MessageId, "error", nackErr)
		}
		return
	}

	if err := delivery.Ack(false); err != nil {
		c.log.Error("failed to ack notification event", "message_id", delivery.MessageId, "error", err)
	}
}

func (c *Consumer) dispatch(ctx context.Context, handler interface {
	HandleSubscriptionConfirmationRequested(context.Context, events.SubscriptionConfirmationRequested) error
	HandleReleaseNotificationRequested(context.Context, events.ReleaseNotificationRequested) error
}, delivery amqp.Delivery,
) error {
	switch delivery.RoutingKey {
	case events.SubscriptionConfirmationRequestedType:
		var event events.SubscriptionConfirmationRequested
		if err := json.Unmarshal(delivery.Body, &event); err != nil {
			return fmt.Errorf("decode subscription confirmation event: %w", err)
		}
		return handler.HandleSubscriptionConfirmationRequested(ctx, event)
	case events.ReleaseNotificationRequestedType:
		var event events.ReleaseNotificationRequested
		if err := json.Unmarshal(delivery.Body, &event); err != nil {
			return fmt.Errorf("decode release notification event: %w", err)
		}
		return handler.HandleReleaseNotificationRequested(ctx, event)
	default:
		return fmt.Errorf("unknown notification event type: %s", delivery.RoutingKey)
	}
}

func (c *Consumer) Close() error {
	return errors.Join(c.ch.Close(), c.conn.Close())
}

func declareTopology(ch *amqp.Channel, cfg Config) error {
	if err := ch.ExchangeDeclare(cfg.Exchange, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare notification exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(cfg.Exchange+".dlx", amqp.ExchangeFanout, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare notification dead-letter exchange: %w", err)
	}

	if _, err := ch.QueueDeclare(cfg.DLQ, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare notification dead-letter queue: %w", err)
	}
	if err := ch.QueueBind(cfg.DLQ, "", cfg.Exchange+".dlx", false, nil); err != nil {
		return fmt.Errorf("bind notification dead-letter queue: %w", err)
	}

	args := amqp.Table{
		"x-dead-letter-exchange": cfg.Exchange + ".dlx",
	}
	if _, err := ch.QueueDeclare(cfg.Queue, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare notification queue: %w", err)
	}

	for _, routingKey := range []string{
		events.SubscriptionConfirmationRequestedType,
		events.ReleaseNotificationRequestedType,
	} {
		if err := ch.QueueBind(cfg.Queue, routingKey, cfg.Exchange, false, nil); err != nil {
			return fmt.Errorf("bind notification queue: %w", err)
		}
	}

	return nil
}

func dial(url string) (*amqp.Connection, error) {
	deadline := time.Now().Add(dialTimeout)
	var lastErr error

	for {
		conn, err := amqp.Dial(url)
		if err == nil {
			return conn, nil
		}
		lastErr = err

		if time.Now().After(deadline) {
			return nil, lastErr
		}
		time.Sleep(dialInterval)
	}
}
