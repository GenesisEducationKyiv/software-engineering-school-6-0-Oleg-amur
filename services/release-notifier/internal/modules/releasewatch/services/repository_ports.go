package services

import "context"

import "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasewatch/models"

type TrackedRepositoryStore interface {
	GetByName(ctx context.Context, name string) (*models.TrackedRepository, error)
	Create(ctx context.Context, name string, lastSeenTag string) (*models.TrackedRepository, error)
}

type RepositoryMetadataClient interface {
	GetRepositoryLatestTag(ctx context.Context, repoAddr string) (string, error)
	CheckIfRepoExists(ctx context.Context, repoAddr string) (bool, error)
}
