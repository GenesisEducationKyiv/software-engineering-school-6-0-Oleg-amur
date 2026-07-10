package postgresql

import (
	"context"
	"database/sql"
	"fmt"

	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/postgresql"
	subscriptiondomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

type SubscriptionSagaStore struct {
	db postgresqladapter.Queryable
}

func NewSubscriptionSagaStore(db postgresqladapter.Queryable) *SubscriptionSagaStore {
	return &SubscriptionSagaStore{db: db}
}

func (r *SubscriptionSagaStore) CompleteSubscriptionConfirmation(ctx context.Context, sagaID int) error {
	query := `
		UPDATE subscription_sagas
		SET saga_status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2 AND saga_status IN ($3, $1)`
	result, err := r.db.ExecContext(
		ctx,
		query,
		subscriptiondomain.SagaStatusCompleted,
		sagaID,
		subscriptiondomain.SagaStatusStarted,
	)
	return validateSagaTransition(result, err, sagaID, subscriptiondomain.SagaStatusCompleted)
}

func (r *SubscriptionSagaStore) CreateStarted(
	ctx context.Context,
	subscriptionID int,
) (*subscriptiondomain.SubscriptionSaga, error) {
	query := `
		INSERT INTO subscription_sagas (subscription_id, saga_status)
		VALUES ($1, $2)
		RETURNING id, subscription_id, saga_status, failure_reason`

	saga := &subscriptiondomain.SubscriptionSaga{}
	err := r.db.QueryRowContext(ctx, query, subscriptionID, subscriptiondomain.SagaStatusStarted).Scan(
		&saga.ID,
		&saga.SubscriptionID,
		&saga.Status,
		&saga.FailureReason,
	)
	if err != nil {
		return nil, err
	}
	return saga, nil
}

func (r *SubscriptionSagaStore) MarkCompensated(
	ctx context.Context,
	sagaID int,
	reason string,
) error {
	query := `
		UPDATE subscription_sagas
		SET saga_status = $1, failure_reason = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3 AND saga_status IN ($4, $1)`
	result, err := r.db.ExecContext(
		ctx,
		query,
		subscriptiondomain.SagaStatusCompensated,
		reason,
		sagaID,
		subscriptiondomain.SagaStatusStarted,
	)
	return validateSagaTransition(result, err, sagaID, subscriptiondomain.SagaStatusCompensated)
}

func validateSagaTransition(
	result sql.Result,
	err error,
	sagaID int,
	target subscriptiondomain.SagaStatus,
) error {
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf(
			"%w: saga %d cannot transition to status %d",
			subscriptiondomain.ErrInvalidSagaTransition,
			sagaID,
			target,
		)
	}
	return nil
}
