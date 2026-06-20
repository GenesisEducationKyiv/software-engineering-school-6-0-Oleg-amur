package domain

import "encoding/json"

const (
	SagaStatusStarted     = 1
	SagaStatusCompleted   = 2
	SagaStatusCompensated = 3

	OutboxStatusPending   = 1
	OutboxStatusPublished = 2
)

type OutboxMessage struct {
	ID        int
	EventType string
	Payload   json.RawMessage
}
