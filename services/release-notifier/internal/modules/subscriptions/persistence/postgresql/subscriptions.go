package postgresql

import (
	"context"
	"database/sql"
	"errors"

	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	releasetrackerdomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/domain"
	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

type SubscriptionRepository struct {
	db postgresqladapter.Queryable
}

func NewSubscriptionRepository(db postgresqladapter.Queryable) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(
	ctx context.Context,
	subID, repoID int,
	token string,
) error {
	query := `
		INSERT INTO subscriptions (subscriber_id, repository_id, subscription_status, token) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (subscriber_id, repository_id) DO NOTHING`

	res, err := r.db.ExecContext(ctx, query, subID, repoID, subscriptionmodels.StatusPending, token)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return apperr.ErrAlreadyExists
	}

	return nil
}

func (r *SubscriptionRepository) GetByToken(
	ctx context.Context,
	token string,
) (*subscriptionmodels.Subscription, error) {
	query := `
		SELECT s.id, s.subscriber_id, s.repository_id, s.subscription_status, s.token, s.created_at, sub.email, repo.name, repo.last_seen_tag
		FROM subscriptions s
		JOIN subscribers sub ON s.subscriber_id = sub.id
		JOIN repositories repo ON s.repository_id = repo.id
		WHERE s.token = $1`

	var s subscriptionmodels.Subscription
	s.Subscriber = &subscriptionmodels.Subscriber{}
	s.Repository = &releasetrackerdomain.Repository{}

	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&s.ID, &s.SubscriberID, &s.RepositoryID, &s.SubscriptionStatus, &s.Token, &s.CreatedAt,
		&s.Subscriber.Email, &s.Repository.Name, &s.Repository.LastSeenTag,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return &s, nil
}

func (r *SubscriptionRepository) Activate(ctx context.Context, token string) error {
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

func (r *SubscriptionRepository) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM subscriptions WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

func (r *SubscriptionRepository) GetActiveByEmail(
	ctx context.Context,
	email string,
) ([]subscriptionmodels.Subscription, error) {
	query := `
		SELECT s.id, s.token, s.subscription_status, sub.email, repo.name, repo.last_seen_tag
		FROM subscriptions s
		JOIN subscribers sub ON s.subscriber_id = sub.id
		JOIN repositories repo ON s.repository_id = repo.id
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
		s.Repository = &releasetrackerdomain.Repository{}

		err := rows.Scan(
			&s.ID,
			&s.Token,
			&s.SubscriptionStatus,
			&s.Subscriber.Email,
			&s.Repository.Name,
			&s.Repository.LastSeenTag,
		)
		if err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return subs, nil
}

func (r *SubscriptionRepository) GetActiveByRepositoryID(
	ctx context.Context,
	repoID int,
) ([]subscriptionmodels.RepositorySubscription, error) {
	query := `
		SELECT s.token, sub.email
		FROM subscriptions s
		JOIN subscribers sub ON s.subscriber_id = sub.id
		WHERE s.repository_id = $1 AND s.subscription_status = $2`

	rows, err := r.db.QueryContext(ctx, query, repoID, subscriptionmodels.StatusActive)
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
