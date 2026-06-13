package rabbitmq

import (
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
)

const (
	dialTimeout  = 30 * time.Second
	dialInterval = 500 * time.Millisecond
)

func DialWithRetry(url string) (*amqp.Connection, error) {
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
