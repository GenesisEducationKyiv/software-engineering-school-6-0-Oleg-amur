package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

type SubscriberRepository struct {
	db postgresqladapter.Queryable
}

func NewSubscriberRepository(db postgresqladapter.Queryable) *SubscriberRepository {
	return &SubscriberRepository{db: db}
}

func (r *SubscriberRepository) Create(ctx context.Context, email string) (*subscriptionmodels.Subscriber, error) {
	query := `
		INSERT INTO subscribers (email) 
		VALUES ($1) 
		RETURNING id, email, created_at`

	var s subscriptionmodels.Subscriber
	err := r.db.QueryRowContext(ctx, query, email).Scan(&s.ID, &s.Email, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}
	return &s, nil
}

func (r *SubscriberRepository) GetByEmail(
	ctx context.Context,
	email string,
) (*subscriptionmodels.Subscriber, error) {
	query := `SELECT id, email, created_at FROM subscribers WHERE email = $1`
	var s subscriptionmodels.Subscriber
	err := r.db.QueryRowContext(ctx, query, email).Scan(&s.ID, &s.Email, &s.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}
