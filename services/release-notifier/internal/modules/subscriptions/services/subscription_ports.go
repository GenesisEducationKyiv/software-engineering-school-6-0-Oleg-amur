package services

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/models"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

type RepositoryTracker interface {
	EnsureTracked(ctx context.Context, repoName string) (*models.TrackedRepositoryRef, error)
}

type SubscriptionRepo interface {
	Create(ctx context.Context, subID, repoID int, token string) error
	Activate(ctx context.Context, token string) error
	DeleteByToken(ctx context.Context, token string) error
	GetActiveByEmail(ctx context.Context, email string) ([]models.Subscription, error)
}

type notificationPublisher interface {
	PublishSubscriptionConfirmation(ctx context.Context, event events.SubscriptionConfirmationRequested) error
}
