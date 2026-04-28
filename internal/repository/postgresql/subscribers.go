package postgresql

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/models"
)

type SubscriberRepository struct {
	db *sql.DB
}

func NewSubscriberRepository(db *sql.DB) *SubscriberRepository {
	return &SubscriberRepository{db: db}
}

func (r *SubscriberRepository) Create(ctx context.Context, email string) (*models.Subscriber, error) {
	query := `
		INSERT INTO subscribers (email) 
		VALUES ($1) 
		RETURNING id, email, created_at`

	var s models.Subscriber
	err := r.db.QueryRowContext(ctx, query, email).Scan(&s.ID, &s.Email, &s.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscriber: %w", err)
	}
	return &s, nil
}

func (r *SubscriberRepository) GetByEmail(ctx context.Context, email string) (*models.Subscriber, error) {
	query := `SELECT id, email, created_at FROM subscribers WHERE email = $1`
	var s models.Subscriber
	err := r.db.QueryRowContext(ctx, query, email).Scan(&s.ID, &s.Email, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
