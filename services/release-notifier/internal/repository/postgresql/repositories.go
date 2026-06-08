package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/model"
)

type RepositoryRepository struct {
	db Queryable
}

func NewRepositoryRepository(db Queryable) *RepositoryRepository {
	return &RepositoryRepository{db: db}
}

func (r *RepositoryRepository) Create(
	ctx context.Context,
	name string,
	lastSeenTag string,
) (*model.Repository, error) {
	query := `
		INSERT INTO repositories (name, last_seen_tag) 
		VALUES ($1, $2) 
		RETURNING id, name, last_seen_tag, created_at`

	var repo model.Repository
	err := r.db.QueryRowContext(ctx, query, name, lastSeenTag).
		Scan(&repo.ID, &repo.Name, &repo.LastSeenTag, &repo.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}
	return &repo, nil
}

func (r *RepositoryRepository) GetByName(
	ctx context.Context,
	name string,
) (*model.Repository, error) {
	query := `SELECT id, name, last_seen_tag, created_at FROM repositories WHERE name = $1`
	var repo model.Repository
	err := r.db.QueryRowContext(ctx, query, name).
		Scan(&repo.ID, &repo.Name, &repo.LastSeenTag, &repo.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrNotFound
		}
		return nil, err
	}
	return &repo, nil
}

func (r *RepositoryRepository) UpdateTag(ctx context.Context, id int, tag string) error {
	query := `UPDATE repositories SET last_seen_tag = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, tag, id)
	return err
}

func (r *RepositoryRepository) GetAll(ctx context.Context) ([]model.Repository, error) {
	query := `SELECT id, name, last_seen_tag, created_at FROM repositories`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		clErr := rows.Close()
		err = errors.Join(err, clErr)
	}()

	var repos []model.Repository
	for rows.Next() {
		var repo model.Repository
		if err := rows.Scan(&repo.ID, &repo.Name, &repo.LastSeenTag, &repo.CreatedAt); err != nil {
			return nil, err
		}
		repos = append(repos, repo)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return repos, nil
}
