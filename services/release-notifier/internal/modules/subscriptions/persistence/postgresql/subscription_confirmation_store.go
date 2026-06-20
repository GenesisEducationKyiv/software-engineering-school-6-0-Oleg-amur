package postgresql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/google/uuid"
)

type SubscriptionConfirmationStore struct {
	db *sql.DB
}

func NewSubscriptionConfirmationStore(db *sql.DB) *SubscriptionConfirmationStore {
	return &SubscriptionConfirmationStore{db: db}
}

func (s *SubscriptionConfirmationStore) StartSubscriptionConfirmation(
	ctx context.Context,
	subscriberID, repositoryID int,
	email string,
	token string,
) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackUnlessCommitted(tx, &err)

	subscriptions := NewSubscriptionRepository(tx)
	sagas := NewSubscriptionSagaRepository(tx)
	outbox := NewOutboxRepository(tx)

	subscriptionID, err := subscriptions.CreateReturningID(ctx, subscriberID, repositoryID, token)
	if err != nil {
		return err
	}

	saga, err := sagas.CreateStarted(ctx, subscriptionID)
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
	if err = outbox.Create(ctx, events.SubscriptionConfirmationRequestedType, event); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func (s *SubscriptionConfirmationStore) CompensateSubscriptionConfirmation(
	ctx context.Context,
	sagaID int,
	subscriptionID int,
	reason string,
) (err error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer rollbackUnlessCommitted(tx, &err)

	if err = NewSubscriptionRepository(tx).DeleteByID(ctx, subscriptionID); err != nil {
		return err
	}
	if err = NewSubscriptionSagaRepository(tx).MarkCompensated(ctx, sagaID, reason); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func rollbackUnlessCommitted(tx *sql.Tx, err *error) {
	if *err == nil {
		return
	}
	rollbackErr := tx.Rollback()
	if rollbackErr != nil && !errors.Is(rollbackErr, sql.ErrTxDone) {
		*err = errors.Join(*err, rollbackErr)
	}
}
