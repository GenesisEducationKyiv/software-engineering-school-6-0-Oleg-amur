package services

import "context"

import "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasewatch/models"

type ScannerRepositoryStore interface {
	GetAll(ctx context.Context) ([]models.TrackedRepository, error)
	UpdateTag(ctx context.Context, id int, tag string) error
}

type ReleaseTagClient interface {
	GetRepositoryLatestTag(ctx context.Context, repoAddr string) (string, error)
}

type ReleaseDetectedHandler interface {
	HandleReleaseDetected(ctx context.Context, event models.ReleaseEvent) error
}

type releaseScanner interface {
	Scan(ctx context.Context)
}
