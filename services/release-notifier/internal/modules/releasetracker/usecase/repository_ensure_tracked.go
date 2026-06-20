package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/domain"
)

type RepositoryStore interface {
	GetByName(ctx context.Context, name string) (*domain.Repository, error)
	Create(ctx context.Context, name string, lastSeenTag string) (*domain.Repository, error)
}

type RepositoryMetadataClient interface {
	GetRepositoryLatestTag(ctx context.Context, repoAddr string) (string, error)
	CheckIfRepoExists(ctx context.Context, repoAddr string) (bool, error)
}

type EnsureRepositoryTracked struct {
	log             *slog.Logger
	repositoryStore RepositoryStore
	githubClient    RepositoryMetadataClient
}

func NewEnsureRepositoryTracked(
	log *slog.Logger,
	store RepositoryStore,
	githubClient RepositoryMetadataClient,
) *EnsureRepositoryTracked {
	return &EnsureRepositoryTracked{
		log:             log,
		repositoryStore: store,
		githubClient:    githubClient,
	}
}

func (s *EnsureRepositoryTracked) Execute(ctx context.Context, repoName string) (*domain.Repository, error) {
	repo, err := s.repositoryStore.GetByName(ctx, repoName)
	if err != nil {
		if !errors.Is(err, apperr.ErrNotFound) {
			return nil, fmt.Errorf("repository check error: %w", err)
		}

		exists, checkErr := s.githubClient.CheckIfRepoExists(ctx, repoName)
		if checkErr != nil {
			if errors.Is(checkErr, apperr.ErrRateLimitExceeded) {
				return nil, apperr.ErrRateLimitExceeded
			}
			return nil, fmt.Errorf("github check existence failed: %w", checkErr)
		}
		if !exists {
			return nil, apperr.ErrRepoNotFound
		}

		tag, tagErr := s.githubClient.GetRepositoryLatestTag(ctx, repoName)
		if tagErr != nil {
			if errors.Is(tagErr, apperr.ErrRateLimitExceeded) {
				return nil, apperr.ErrRateLimitExceeded
			}
			if !errors.Is(tagErr, apperr.ErrRepoNotFound) {
				return nil, fmt.Errorf("github get tag failed: %w", tagErr)
			}
		}

		repo, err = s.repositoryStore.Create(ctx, repoName, tag)
		if err != nil {
			return nil, fmt.Errorf("failed to create repository: %w", err)
		}
	}

	return repo, nil
}
