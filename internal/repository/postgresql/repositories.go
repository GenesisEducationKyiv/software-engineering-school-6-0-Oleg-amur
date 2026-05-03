package postgresql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/models"
)

type RepositoryRepository struct {
	db *sql.DB
}

func NewRepositoryRepository(db *sql.DB) *RepositoryRepository {
	return &RepositoryRepository{db: db}
}

func (r *RepositoryRepository) Create(
	ctx context.Context,
	name string,
	lastSeenTag string,
) (*models.Repository, error) {
	query := `
		INSERT INTO repositories (name, last_seen_tag) 
		VALUES ($1, $2) 
		RETURNING id, name, last_seen_tag, created_at`

	var repo models.Repository
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
) (*models.Repository, error) {
	query := `SELECT id, name, last_seen_tag, created_at FROM repositories WHERE name = $1`
	var repo models.Repository
	err := r.db.QueryRowContext(ctx, query, name).
		Scan(&repo.ID, &repo.Name, &repo.LastSeenTag, &repo.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &repo, nil
}

func (r *RepositoryRepository) UpdateTag(ctx context.Context, id int, tag string) error {
	query := `UPDATE repositories SET last_seen_tag = $1 WHERE id = $2`
	_, err := r.db.ExecContext(ctx, query, tag, id)
	return err
}

func (r *RepositoryRepository) GetAll(ctx context.Context) ([]models.Repository, error) {
	query := `SELECT id, name, last_seen_tag, created_at FROM repositories`
	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer func() {
		clErr := rows.Close()
		err = errors.Join(err, clErr)
	}()

	var repos []models.Repository
	for rows.Next() {
		var repo models.Repository
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
