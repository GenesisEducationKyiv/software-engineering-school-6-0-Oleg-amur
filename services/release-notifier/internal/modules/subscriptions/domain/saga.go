package domain

import (
	"encoding/json"
	"errors"
)

var ErrInvalidSagaTransition = errors.New("invalid saga transition")

type (
	SagaStatus   int
	OutboxStatus int
)

const (
	SagaStatusStarted     SagaStatus = 1
	SagaStatusCompleted   SagaStatus = 2
	SagaStatusCompensated SagaStatus = 3

	OutboxStatusPending   OutboxStatus = 1
	OutboxStatusPublished OutboxStatus = 2
)

type SubscriptionSaga struct {
	ID             int
	SubscriptionID int
	Status         SagaStatus
	FailureReason  *string
}

type OutboxMessage struct {
	ID        int
	EventType string
	Payload   json.RawMessage
}
