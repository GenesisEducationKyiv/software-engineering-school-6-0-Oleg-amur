package postgresql

import (
	"context"
	"database/sql"
	"errors"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	subscriptiondomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
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

	subscriptionID, err := createPendingSubscription(ctx, tx, subscriberID, repositoryID, token)
	if err != nil {
		return err
	}

	saga, err := createSubscriptionSaga(ctx, tx, subscriptionID)
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
	if err = insertOutboxMessage(ctx, tx, events.SubscriptionConfirmationRequestedType, event); err != nil {
		return err
	}

	err = tx.Commit()
	return err
}

func createPendingSubscription(
	ctx context.Context,
	tx *sql.Tx,
	subscriberID, repositoryID int,
	token string,
) (int, error) {
	query := `
		INSERT INTO subscriptions (subscriber_id, repository_id, subscription_status, token)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (subscriber_id, repository_id) DO NOTHING
		RETURNING id`

	var subscriptionID int
	err := tx.QueryRowContext(
		ctx,
		query,
		subscriberID,
		repositoryID,
		subscriptiondomain.StatusPending,
		token,
	).Scan(&subscriptionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, apperr.ErrAlreadyExists
		}
		return 0, err
	}
	return subscriptionID, nil
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
