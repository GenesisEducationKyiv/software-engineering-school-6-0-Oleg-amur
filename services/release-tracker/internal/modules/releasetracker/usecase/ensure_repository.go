package usecase

import (
	"context"
	"errors"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/domain"
)

type EnsureRepository struct {
	repositories repositoryStore
	github       githubClient
}

func NewEnsureRepository(repositories repositoryStore, github githubClient) *EnsureRepository {
	return &EnsureRepository{repositories: repositories, github: github}
}

func (u *EnsureRepository) Execute(ctx context.Context, name string) (*domain.Repository, error) {
	tracked, err := u.repositories.GetByName(ctx, name)
	if err == nil {
		return tracked, nil
	}
	if !errors.Is(err, apperr.ErrRepositoryNotFound) {
		return nil, err
	}

	exists, err := u.github.RepositoryExists(ctx, name)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, apperr.ErrRepositoryNotFound
	}

	tag, err := u.github.LatestTag(ctx, name)
	if err != nil && !errors.Is(err, apperr.ErrRepositoryNotFound) {
		return nil, err
	}
	return u.repositories.Create(ctx, name, tag)
}
