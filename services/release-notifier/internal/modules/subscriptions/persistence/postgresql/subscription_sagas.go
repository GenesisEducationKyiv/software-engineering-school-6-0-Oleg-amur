package postgresql

import (
	"context"

	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/postgresql"
	subscriptiondomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

type SubscriptionSagaRepository struct {
	db postgresqladapter.Queryable
}

func NewSubscriptionSagaRepository(db postgresqladapter.Queryable) *SubscriptionSagaRepository {
	return &SubscriptionSagaRepository{db: db}
}

func (r *SubscriptionSagaRepository) CompleteSubscriptionConfirmation(ctx context.Context, sagaID int) error {
	query := `
		UPDATE subscription_sagas
		SET saga_status = $1, updated_at = CURRENT_TIMESTAMP
		WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, subscriptiondomain.SagaStatusCompleted, sagaID)
	return err
}

func (r *SubscriptionSagaRepository) CreateStarted(
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

func (r *SubscriptionSagaRepository) MarkCompensated(
	ctx context.Context,
	sagaID int,
	reason string,
) error {
	query := `
		UPDATE subscription_sagas
		SET saga_status = $1, failure_reason = $2, updated_at = CURRENT_TIMESTAMP
		WHERE id = $3`
	_, err := r.db.ExecContext(ctx, query, subscriptiondomain.SagaStatusCompensated, reason, sagaID)
	return err
}
