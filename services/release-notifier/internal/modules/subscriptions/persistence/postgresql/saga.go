package postgresql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	subscriptiondomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/google/uuid"
)

type SagaStore struct {
	db *sql.DB
}

func NewSagaStore(db *sql.DB) *SagaStore {
	return &SagaStore{db: db}
}

func (s *SagaStore) StartSubscriptionConfirmation(
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

	sagaID, err := createSubscriptionSaga(ctx, tx, subscriptionID)
	if err != nil {
		return err
	}

	event := events.SubscriptionConfirmationRequested{
		EventID:           uuid.NewString(),
		SchemaVersion:     events.NotificationSchemaVersion,
		SagaID:            sagaID,
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

func (s *SagaStore) FetchPendingOutbox(ctx context.Context, limit int) ([]subscriptiondomain.OutboxMessage, error) {
	query := `
		SELECT id, event_type, payload
		FROM outbox_messages
		WHERE outbox_status = $1
		ORDER BY created_at
		LIMIT $2`

	rows, err := s.db.QueryContext(ctx, query, subscriptiondomain.OutboxStatusPending, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []subscriptiondomain.OutboxMessage
	for rows.Next() {
		var message subscriptiondomain.OutboxMessage
		if err := rows.Scan(&message.ID, &message.EventType, &message.Payload); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return messages, nil
}

func (s *SagaStore) MarkOutboxPublished(ctx context.Context, id int) error {
	query := `
		UPDATE outbox_messages
		SET outbox_status = $1, published_at = CURRENT_TIMESTAMP
		WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, subscriptiondomain.OutboxStatusPublished, id)
	return err
}

func (s *SagaStore) MarkOutboxPublishFailed(ctx context.Context, id int, cause error) error {
	query := `
		UPDATE outbox_messages
		SET attempts = attempts + 1, last_error = $1
		WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, cause.Error(), id)
	return err
}

func (s *SagaStore) CompleteSubscriptionConfirmation(ctx context.Context, sagaID int) error {
	query := `
		UPDATE subscription_sagas
		SET saga_status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, subscriptiondomain.SagaStatusCompleted, sagaID)
	return err
}

func (s *SagaStore) CompensateSubscriptionConfirmation(
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

	deleteQuery := `DELETE FROM subscriptions WHERE id = $1 AND subscription_status = $2`
	if _, err = tx.ExecContext(ctx, deleteQuery, subscriptionID, subscriptiondomain.StatusPending); err != nil {
		return err
	}

	updateQuery := `
		UPDATE subscription_sagas
		SET saga_status = $1, failure_reason = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3`
	if _, err = tx.ExecContext(ctx, updateQuery, subscriptiondomain.SagaStatusCompensated, reason, sagaID); err != nil {
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

func createSubscriptionSaga(ctx context.Context, tx *sql.Tx, subscriptionID int) (int, error) {
	query := `
		INSERT INTO subscription_sagas (subscription_id, saga_status)
		VALUES ($1, $2)
		RETURNING id`

	var sagaID int
	err := tx.QueryRowContext(ctx, query, subscriptionID, subscriptiondomain.SagaStatusStarted).Scan(&sagaID)
	return sagaID, err
}

func insertOutboxMessage(ctx context.Context, tx *sql.Tx, eventType string, payload any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal outbox payload: %w", err)
	}

	query := `
		INSERT INTO outbox_messages (event_type, payload, outbox_status)
		VALUES ($1, $2, $3)`
	_, err = tx.ExecContext(ctx, query, eventType, body, subscriptiondomain.OutboxStatusPending)
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
