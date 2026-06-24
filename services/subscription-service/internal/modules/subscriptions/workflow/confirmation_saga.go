package workflow

import (
	"context"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/google/uuid"
)

type TransactionManager interface {
	Run(ctx context.Context, work func(context.Context) error) error
}

type SubscriptionStore interface {
	Create(ctx context.Context, subscriberID int, repositoryName, token string) (int, error)
	DeleteByID(ctx context.Context, subscriptionID int) error
}

type SubscriptionSagaStore interface {
	CreateStarted(ctx context.Context, subscriptionID int) (*domain.SubscriptionSaga, error)
	CompleteSubscriptionConfirmation(ctx context.Context, sagaID int) error
	MarkCompensated(ctx context.Context, sagaID int, reason string) error
}

type SubscriptionOutboxStore interface {
	Create(ctx context.Context, eventType string, payload any) error
}

type SubscriptionConfirmationSaga struct {
	log                *slog.Logger
	transactionManager TransactionManager
	subscriptions      SubscriptionStore
	sagas              SubscriptionSagaStore
	outbox             SubscriptionOutboxStore
}

func NewSubscriptionConfirmationSaga(
	log *slog.Logger,
	transactionManager TransactionManager,
	subscriptions SubscriptionStore,
	sagas SubscriptionSagaStore,
	outbox SubscriptionOutboxStore,
) *SubscriptionConfirmationSaga {
	return &SubscriptionConfirmationSaga{
		log:                log,
		transactionManager: transactionManager,
		subscriptions:      subscriptions,
		sagas:              sagas,
		outbox:             outbox,
	}
}

func (s *SubscriptionConfirmationSaga) StartSubscriptionConfirmation(
	ctx context.Context,
	subscriberID int,
	repositoryName, email string,
	token string,
) error {
	return s.transactionManager.Run(ctx, func(txCtx context.Context) error {
		subscriptionID, err := s.subscriptions.Create(
			txCtx,
			subscriberID,
			repositoryName,
			token,
		)
		if err != nil {
			return err
		}

		saga, err := s.sagas.CreateStarted(txCtx, subscriptionID)
		if err != nil {
			return err
		}

		event := events.SubscriptionConfirmationRequested{
			EventID:           uuid.NewString(),
			SchemaVersion:     events.NotificationSchemaVersion,
			SagaID:            saga.ID,
			SubscriptionID:    subscriptionID,
			Email:             email,
			ConfirmationToken: token,
		}
		return s.outbox.Create(txCtx, events.SubscriptionConfirmationRequestedType, event)
	})
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
	return s.sagas.CompleteSubscriptionConfirmation(ctx, event.SagaID)
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
	return s.transactionManager.Run(ctx, func(txCtx context.Context) error {
		if err := s.subscriptions.DeleteByID(txCtx, event.SubscriptionID); err != nil {
			return err
		}
		return s.sagas.MarkCompensated(txCtx, event.SagaID, event.Reason)
	})
}
