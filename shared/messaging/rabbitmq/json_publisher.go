package rabbitmq

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	contentTypeJSON = "application/json"
	publishTimeout  = 5 * time.Second
)

type JSONPublisher struct {
	ch        *amqp.Channel
	exchange  string
	confirms  <-chan amqp.Confirmation
	publishMu sync.Mutex
}

func NewJSONPublisher(ch *amqp.Channel, exchange string) (*JSONPublisher, error) {
	if err := ch.Confirm(false); err != nil {
		return nil, fmt.Errorf("enable publisher confirms: %w", err)
	}

	return &JSONPublisher{
		ch:       ch,
		exchange: exchange,
		confirms: ch.NotifyPublish(make(chan amqp.Confirmation, 1)),
	}, nil
}

func (p *JSONPublisher) Publish(
	ctx context.Context,
	routingKey string,
	messageID string,
	payload any,
) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
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
			MessageId:    messageID,
			Timestamp:    time.Now().UTC(),
			Body:         body,
		},
	); err != nil {
		return fmt.Errorf("publish message: %w", err)
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
