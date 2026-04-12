package postgresql

import (
	"context"
	"database/sql"

	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/apperr"
	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/models"
)

type SubscriptionRepository struct {
	db *sql.DB
}

func NewSubscriptionRepository(db *sql.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

func (r *SubscriptionRepository) Create(ctx context.Context, subID, repoID int, token string) error {
	query := `
		INSERT INTO subscriptions (subscriber_id, repository_id, subscription_status, token) 
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (subscriber_id, repository_id) DO NOTHING`

	res, err := r.db.ExecContext(ctx, query, subID, repoID, models.StatusPending, token)
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

func (r *SubscriptionRepository) GetByToken(ctx context.Context, token string) (*models.Subscription, error) {
	query := `
		SELECT s.id, s.subscriber_id, s.repository_id, s.subscription_status, s.token, s.created_at, sub.email, repo.name, repo.last_seen_tag
		FROM subscriptions s
		JOIN subscribers sub ON s.subscriber_id = sub.id
		JOIN repositories repo ON s.repository_id = repo.id
		WHERE s.token = $1`

	var s models.Subscription
	s.Subscriber = &models.Subscriber{}
	s.Repository = &models.Repository{}

	err := r.db.QueryRowContext(ctx, query, token).Scan(
		&s.ID, &s.SubscriberID, &s.RepositoryID, &s.SubscriptionStatus, &s.Token, &s.CreatedAt,
		&s.Subscriber.Email, &s.Repository.Name, &s.Repository.LastSeenTag,
	)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SubscriptionRepository) Activate(ctx context.Context, token string) error {
	query := `UPDATE subscriptions SET subscription_status = $1 WHERE token = $2`
	result, err := r.db.ExecContext(ctx, query, models.StatusActive, token)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *SubscriptionRepository) DeleteByToken(ctx context.Context, token string) error {
	query := `DELETE FROM subscriptions WHERE token = $1`
	_, err := r.db.ExecContext(ctx, query, token)
	return err
}

func (r *SubscriptionRepository) GetActiveByEmail(ctx context.Context, email string) ([]models.Subscription, error) {
	query := `
		SELECT s.id, s.token, s.subscription_status, sub.email, repo.name, repo.last_seen_tag
		FROM subscriptions s
		JOIN subscribers sub ON s.subscriber_id = sub.id
		JOIN repositories repo ON s.repository_id = repo.id
		WHERE sub.email = $1 AND s.subscription_status = $2`

	rows, err := r.db.QueryContext(ctx, query, email, models.StatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.Subscription
	for rows.Next() {
		var s models.Subscription
		s.Subscriber = &models.Subscriber{}
		s.Repository = &models.Repository{}

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

	return subs, nil
}

func (r *SubscriptionRepository) GetActiveByRepoID(ctx context.Context, repoID int) ([]models.Subscription, error) {
	query := `
		SELECT s.id, s.token, s.subscription_status, sub.email
		FROM subscriptions s
		JOIN subscribers sub ON s.subscriber_id = sub.id
		WHERE s.repository_id = $1 AND s.subscription_status = $2`

	rows, err := r.db.QueryContext(ctx, query, repoID, models.StatusActive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []models.Subscription
	for rows.Next() {
		var s models.Subscription
		s.Subscriber = &models.Subscriber{}

		if err := rows.Scan(&s.ID, &s.Token, &s.SubscriptionStatus, &s.Subscriber.Email); err != nil {
			return nil, err
		}
		subs = append(subs, s)
	}
	return subs, nil
}
