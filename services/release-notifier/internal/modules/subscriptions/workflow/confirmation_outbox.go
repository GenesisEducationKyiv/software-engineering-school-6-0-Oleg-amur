package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

const (
	subscriptionOutboxBatchSize   = 50
	subscriptionOutboxMaxAttempts = 5
)

type OutboxStore interface {
	FetchPendingOutbox(ctx context.Context, limit int) ([]domain.OutboxMessage, error)
	MarkOutboxPublished(ctx context.Context, id int) error
	RecordOutboxFailure(ctx context.Context, id int, cause error, maxAttempts int) error
}

type SubscriptionConfirmationPublisher interface {
	PublishSubscriptionConfirmation(ctx context.Context, event events.SubscriptionConfirmationRequested) error
}

type PublishSubscriptionOutbox struct {
	log       *slog.Logger
	store     OutboxStore
	publisher SubscriptionConfirmationPublisher
}

func NewPublishSubscriptionOutbox(
	log *slog.Logger,
	store OutboxStore,
	publisher SubscriptionConfirmationPublisher,
) *PublishSubscriptionOutbox {
	return &PublishSubscriptionOutbox{
		log:       log,
		store:     store,
		publisher: publisher,
	}
}

func (p *PublishSubscriptionOutbox) Execute(ctx context.Context) {
	messages, err := p.store.FetchPendingOutbox(ctx, subscriptionOutboxBatchSize)
	if err != nil {
		p.log.Error("failed to fetch subscription outbox", "err", err)
		return
	}

	for _, message := range messages {
		event, err := decodeSubscriptionConfirmation(message)
		if err != nil {
			p.log.Error("invalid subscription outbox message", "outbox_id", message.ID, "err", err)
			p.recordFailure(ctx, message.ID, err, 1)
			continue
		}

		if err := p.publisher.PublishSubscriptionConfirmation(ctx, event); err != nil {
			p.log.Error("failed to publish subscription outbox message", "outbox_id", message.ID, "err", err)
			p.recordFailure(ctx, message.ID, err, subscriptionOutboxMaxAttempts)
			continue
		}

		if err := p.store.MarkOutboxPublished(ctx, message.ID); err != nil {
			p.log.Error("failed to mark subscription outbox message as published", "outbox_id", message.ID, "err", err)
			p.recordFailure(ctx, message.ID, err, subscriptionOutboxMaxAttempts)
		}
	}
}

func decodeSubscriptionConfirmation(message domain.OutboxMessage) (events.SubscriptionConfirmationRequested, error) {
	if message.EventType != events.SubscriptionConfirmationRequestedType {
		return events.SubscriptionConfirmationRequested{}, fmt.Errorf(
			"unsupported subscription outbox event type: %s",
			message.EventType,
		)
	}

	var event events.SubscriptionConfirmationRequested
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		return events.SubscriptionConfirmationRequested{}, fmt.Errorf("decode subscription outbox payload: %w", err)
	}
	return event, nil
}

func (p *PublishSubscriptionOutbox) recordFailure(
	ctx context.Context,
	id int,
	cause error,
	maxAttempts int,
) {
	if err := p.store.RecordOutboxFailure(ctx, id, cause, maxAttempts); err != nil {
		p.log.Error("failed to record subscription outbox failure", "outbox_id", id, "err", err)
	}
}
