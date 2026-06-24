package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/domain"
)

type GetRepository struct {
	repositories repositoryStore
}

func NewGetRepository(repositories repositoryStore) *GetRepository {
	return &GetRepository{repositories: repositories}
}

func (u *GetRepository) Execute(ctx context.Context, name string) (*domain.Repository, error) {
	return u.repositories.GetByName(ctx, name)
}
