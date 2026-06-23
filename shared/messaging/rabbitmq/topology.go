package rabbitmq

import (
	"fmt"

	amqp "github.com/rabbitmq/amqp091-go"
)

func DeclareTopology(ch *amqp.Channel, cfg Config, routingKeys []string) error {
	if err := ch.ExchangeDeclare(cfg.Exchange, amqp.ExchangeDirect, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	if err := ch.ExchangeDeclare(cfg.Exchange+".dlx", amqp.ExchangeFanout, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dead-letter exchange: %w", err)
	}

	if _, err := ch.QueueDeclare(cfg.DLQ, true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dead-letter queue: %w", err)
	}
	if err := ch.QueueBind(cfg.DLQ, "", cfg.Exchange+".dlx", false, nil); err != nil {
		return fmt.Errorf("bind dead-letter queue: %w", err)
	}

	args := amqp.Table{
		"x-dead-letter-exchange": cfg.Exchange + ".dlx",
	}
	if _, err := ch.QueueDeclare(cfg.Queue, true, false, false, false, args); err != nil {
		return fmt.Errorf("declare queue: %w", err)
	}

	for _, routingKey := range routingKeys {
		if err := ch.QueueBind(cfg.Queue, routingKey, cfg.Exchange, false, nil); err != nil {
			return fmt.Errorf("bind queue: %w", err)
		}
	}

	return nil
}
