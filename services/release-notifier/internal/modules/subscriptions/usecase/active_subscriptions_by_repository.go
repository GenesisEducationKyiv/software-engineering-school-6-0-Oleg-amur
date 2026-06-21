package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

type ActiveSubscriptionsByRepositoryStore interface {
	GetActiveByRepositoryName(ctx context.Context, repoName string) ([]domain.RepositorySubscription, error)
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
	repoName string,
) ([]domain.RepositorySubscription, error) {
	return u.subscriptions.GetActiveByRepositoryName(ctx, repoName)
}
