package workflow

import (
	"context"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

type SubscriptionSagaStore interface {
	CompleteSubscriptionConfirmation(ctx context.Context, sagaID int) error
	CompensateSubscriptionConfirmation(ctx context.Context, sagaID int, subscriptionID int, reason string) error
}

type SubscriptionConfirmationSaga struct {
	log   *slog.Logger
	store SubscriptionSagaStore
}

func NewSubscriptionConfirmationSaga(log *slog.Logger, store SubscriptionSagaStore) *SubscriptionConfirmationSaga {
	return &SubscriptionConfirmationSaga{
		log:   log,
		store: store,
	}
}

func (s *SubscriptionConfirmationSaga) HandleSubscriptionConfirmationSucceeded(
	ctx context.Context,
	event events.SubscriptionConfirmationSucceeded,
) error {
	s.log.Info(
		"subscription confirmation saga completed",
		"saga_id",
		event.SagaID,
		"subscription_id",
		event.SubscriptionID,
		"email",
		event.Email,
	)
	return s.store.CompleteSubscriptionConfirmation(ctx, event.SagaID)
}

func (s *SubscriptionConfirmationSaga) HandleSubscriptionConfirmationFailed(
	ctx context.Context,
	event events.SubscriptionConfirmationFailed,
) error {
	s.log.Warn(
		"subscription confirmation saga failed; compensating pending subscription",
		"saga_id",
		event.SagaID,
		"subscription_id",
		event.SubscriptionID,
		"email",
		event.Email,
		"reason",
		event.Reason,
	)
	return s.store.CompensateSubscriptionConfirmation(ctx, event.SagaID, event.SubscriptionID, event.Reason)
}
