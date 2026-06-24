package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/domain"
)

type RepositoryStore struct {
	db *sql.DB
}

func NewRepositoryStore(db *sql.DB) *RepositoryStore { return &RepositoryStore{db: db} }

func (s *RepositoryStore) Create(ctx context.Context, name, lastSeenTag string) (*domain.Repository, error) {
	const query = `
		INSERT INTO repositories (name, last_seen_tag)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET name = EXCLUDED.name
		RETURNING id, name, last_seen_tag, created_at`

	var repository domain.Repository
	if err := s.db.QueryRowContext(ctx, query, name, lastSeenTag).Scan(
		&repository.ID,
		&repository.Name,
		&repository.LastSeenTag,
		&repository.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("create repository: %w", err)
	}
	return &repository, nil
}

func (s *RepositoryStore) GetByName(ctx context.Context, name string) (*domain.Repository, error) {
	const query = `SELECT id, name, last_seen_tag, created_at FROM repositories WHERE name = $1`
	var repository domain.Repository
	if err := s.db.QueryRowContext(ctx, query, name).Scan(
		&repository.ID,
		&repository.Name,
		&repository.LastSeenTag,
		&repository.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, apperr.ErrRepositoryNotFound
		}
		return nil, fmt.Errorf("get repository: %w", err)
	}
	return &repository, nil
}

func (s *RepositoryStore) GetAll(ctx context.Context) (_ []domain.Repository, err error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, last_seen_tag, created_at FROM repositories`)
	if err != nil {
		return nil, fmt.Errorf("list repositories: %w", err)
	}
	defer func() { err = errors.Join(err, rows.Close()) }()

	repositories := make([]domain.Repository, 0)
	for rows.Next() {
		var repository domain.Repository
		if err := rows.Scan(
			&repository.ID,
			&repository.Name,
			&repository.LastSeenTag,
			&repository.CreatedAt,
		); err != nil {
			return nil, err
		}
		repositories = append(repositories, repository)
	}
	return repositories, rows.Err()
}

func (s *RepositoryStore) UpdateTag(ctx context.Context, id int, tag string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE repositories SET last_seen_tag = $1 WHERE id = $2`, tag, id)
	return err
}
