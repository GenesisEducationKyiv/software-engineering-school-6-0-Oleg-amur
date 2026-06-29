package postgresql

import (
	"context"
	"database/sql"
	"errors"

	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/adapters/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/apperr"
	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
)

type SubscriptionStore struct {
	db postgresqladapter.Queryable
}

func NewSubscriptionStore(db postgresqladapter.Queryable) *SubscriptionStore {
	return &SubscriptionStore{db: db}
}

func (r *SubscriptionStore) Create(
	ctx context.Context,
	subID int,
	repositoryID int64,
	token string,
) (int, error) {
	query := `
		INSERT INTO subscriptions (subscriber_id, repository_id, subscription_status, token)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (subscriber_id, repository_id) DO NOTHING
		RETURNING id`

	var subscriptionID int
	err := r.db.QueryRowContext(ctx, query, subID, repositoryID, subscriptionmodels.StatusPending, token).
		Scan(&subscriptionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, apperr.ErrAlreadyExists
		}
		return 0, err
	}

	return subscriptionID, nil
}

func (r *SubscriptionStore) GetByToken(
	ctx context.Context,
	token string,
) (*subscriptionmodels.Subscription, error) {
	query := `
		SELECT s.id, s.subscriber_id, s.repository_id, s.subscription_status, s.token, s.created_at, sub.email
		FROM subscriptions s
		JOIN subscribers sub ON s.subscriber_id = sub.id
		WHERE s.token = $1`

	var s subscriptionmodels.Subscription
	s.Subscriber = &subscriptionmodels.Subscriber{}
	s.Repository = &subscriptionmodels.Repository{}

	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&s.ID, &s.SubscriberID, &s.RepositoryID, &s.SubscriptionStatus, &s.Token, &s.CreatedAt,
		&s.Subscriber.Email,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	s.Repository.ID = s.RepositoryID
	return &s, nil
}

func (r *SubscriptionStore) Activate(ctx context.Context, token string) error {
	query := `UPDATE subscriptions SET subscription_status = $1 WHERE token = $2`
	result, err := r.db.ExecContext(ctx, query, subscriptionmodels.StatusActive, token)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *SubscriptionStore) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM subscriptions WHERE token = $1`
	result, err := r.db.ExecContext(ctx, query, token)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return apperr.ErrNotFound
	}
	return nil
}

func (r *SubscriptionStore) DeleteByID(ctx context.Context, id int) error {
	query := `DELETE FROM subscriptions WHERE id = $1`
	_, err := r.db.ExecContext(ctx, query, id)
	return err
}

func (r *SubscriptionStore) GetActiveByEmail(
	ctx context.Context,
	email string,
) ([]subscriptionmodels.Subscription, error) {
	query := `
		SELECT s.id, s.token, s.subscription_status, sub.email, s.repository_id
		FROM subscriptions s
		JOIN subscribers sub ON s.subscriber_id = sub.id
		WHERE sub.email = $1 AND s.subscription_status = $2`

	rows, err := r.db.QueryContext(ctx, query, email, subscriptionmodels.StatusActive)
	if err != nil {
		return nil, err
	}
	defer func() {
		clErr := rows.Close()
		err = errors.Join(err, clErr)
	}()

	var subs []subscriptionmodels.Subscription
	for rows.Next() {
		var s subscriptionmodels.Subscription
		s.Subscriber = &subscriptionmodels.Subscriber{}
		s.Repository = &subscriptionmodels.Repository{}

		err := rows.Scan(
			&s.ID,
			&s.Token,
			&s.SubscriptionStatus,
			&s.Subscriber.Email,
			&s.RepositoryID,
		)
		if err != nil {
			return nil, err
		}
		s.Repository.ID = s.RepositoryID
		subs = append(subs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subs, nil
}

func (r *SubscriptionStore) GetActiveByRepositoryID(
	ctx context.Context,
	repositoryID int64,
) ([]subscriptionmodels.RepositorySubscription, error) {
	query := `
		SELECT s.token, sub.email
		FROM subscriptions s
		JOIN subscribers sub ON s.subscriber_id = sub.id
		WHERE s.repository_id = $1 AND s.subscription_status = $2`

	rows, err := r.db.QueryContext(ctx, query, repositoryID, subscriptionmodels.StatusActive)
	if err != nil {
		return nil, err
	}
	defer func() {
		clErr := rows.Close()
		err = errors.Join(err, clErr)
	}()

	var subscriptions []subscriptionmodels.RepositorySubscription
	for rows.Next() {
		var subscription subscriptionmodels.RepositorySubscription

		if err := rows.Scan(
			&subscription.UnsubscribeToken,
			&subscription.Email,
		); err != nil {
			return nil, err
		}
		subscriptions = append(subscriptions, subscription)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subscriptions, nil
}
