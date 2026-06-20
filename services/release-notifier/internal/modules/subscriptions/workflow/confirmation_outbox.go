package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

const subscriptionOutboxBatchSize = 50

type OutboxStore interface {
	FetchPendingOutbox(ctx context.Context, limit int) ([]domain.OutboxMessage, error)
	MarkOutboxPublished(ctx context.Context, id int) error
	MarkOutboxPublishFailed(ctx context.Context, id int, cause error) error
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
		if err := p.publish(ctx, message); err != nil {
			p.log.Error("failed to publish subscription outbox message", "outbox_id", message.ID, "err", err)
			if markErr := p.store.MarkOutboxPublishFailed(ctx, message.ID, err); markErr != nil {
				p.log.Error("failed to mark subscription outbox failure", "outbox_id", message.ID, "err", markErr)
			}
		}
	}
}

func (p *PublishSubscriptionOutbox) publish(ctx context.Context, message domain.OutboxMessage) error {
	if message.EventType != events.SubscriptionConfirmationRequestedType {
		return fmt.Errorf("unsupported subscription outbox event type: %s", message.EventType)
	}

	var event events.SubscriptionConfirmationRequested
	if err := json.Unmarshal(message.Payload, &event); err != nil {
		return fmt.Errorf("decode subscription outbox payload: %w", err)
	}
	if err := p.publisher.PublishSubscriptionConfirmation(ctx, event); err != nil {
		return err
	}
	return p.store.MarkOutboxPublished(ctx, message.ID)
}
