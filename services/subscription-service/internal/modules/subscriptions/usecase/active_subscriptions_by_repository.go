package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
)

type ActiveSubscriptionsByRepositoryStore interface {
	GetActiveByRepositoryID(ctx context.Context, repositoryID int64) ([]domain.RepositorySubscription, error)
}

type ListActiveSubscriptionsByRepository struct {
	subscriptions ActiveSubscriptionsByRepositoryStore
}

func NewListActiveSubscriptionsByRepository(
	subscriptions ActiveSubscriptionsByRepositoryStore,
) *ListActiveSubscriptionsByRepository {
	return &ListActiveSubscriptionsByRepository{subscriptions: subscriptions}
}

func (u *ListActiveSubscriptionsByRepository) Execute(
	ctx context.Context,
	repositoryID int64,
) ([]domain.RepositorySubscription, error) {
	return u.subscriptions.GetActiveByRepositoryID(ctx, repositoryID)
}
