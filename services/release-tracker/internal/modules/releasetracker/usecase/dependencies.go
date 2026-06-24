package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

type repositoryStore interface {
	Create(context.Context, string, string) (*domain.Repository, error)
	GetByName(context.Context, string) (*domain.Repository, error)
	GetByID(context.Context, int64) (*domain.Repository, error)
	GetAll(context.Context) ([]domain.Repository, error)
	UpdateTag(context.Context, int64, string) error
}

type githubClient interface {
	RepositoryExists(context.Context, string) (bool, error)
	LatestTag(context.Context, string) (string, error)
}

type subscriptionClient interface {
	ListActiveByRepository(context.Context, int64) ([]domain.ActiveSubscription, error)
}

type notificationPublisher interface {
	Publish(context.Context, events.ReleaseNotificationRequested) error
}
