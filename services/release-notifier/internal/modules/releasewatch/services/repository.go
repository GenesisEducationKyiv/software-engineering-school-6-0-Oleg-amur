package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasewatch/models"
)

type RepositoryService struct {
	log            *slog.Logger
	repositoryRepo TrackedRepositoryStore
	githubClient   RepositoryMetadataClient
}

func NewRepositoryService(
	log *slog.Logger,
	repo TrackedRepositoryStore,
	githubClient RepositoryMetadataClient,
) *RepositoryService {
	return &RepositoryService{
		log:            log,
		repositoryRepo: repo,
		githubClient:   githubClient,
	}
}

func (s *RepositoryService) EnsureTracked(ctx context.Context, repoName string) (*models.TrackedRepository, error) {
	repo, err := s.repositoryRepo.GetByName(ctx, repoName)
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

		repo, err = s.repositoryRepo.Create(ctx, repoName, tag)
		if err != nil {
			return nil, fmt.Errorf("failed to create repository: %w", err)
		}
	}

	return repo, nil
}
