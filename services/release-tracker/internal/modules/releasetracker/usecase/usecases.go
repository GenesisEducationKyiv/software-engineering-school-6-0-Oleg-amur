package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/domain"
)

type Usecases struct {
	EnsureRepository *EnsureRepository
	RepositoryQuery  *GetRepository
	ScanRepositories *ScanRepositories
}

func (u Usecases) EnsureTracked(ctx context.Context, name string) (*domain.Repository, error) {
	return u.EnsureRepository.Execute(ctx, name)
}

func (u Usecases) GetRepository(ctx context.Context, name string) (*domain.Repository, error) {
	return u.RepositoryQuery.Execute(ctx, name)
}
