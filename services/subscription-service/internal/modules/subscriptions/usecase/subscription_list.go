package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
)

type ActiveSubscriptionStore interface {
	GetActiveByEmail(ctx context.Context, email string) ([]domain.Subscription, error)
}

type ListSubscriptions struct {
	subscriptions ActiveSubscriptionStore
	repositories  RepositoryMetadataReader
}

type RepositoryMetadataReader interface {
	GetRepository(ctx context.Context, id int64) (*RepositoryView, error)
}

func NewListSubscriptions(
	subscriptions ActiveSubscriptionStore,
	repositories RepositoryMetadataReader,
) *ListSubscriptions {
	return &ListSubscriptions{subscriptions: subscriptions, repositories: repositories}
}

func (u *ListSubscriptions) Execute(
	ctx context.Context,
	email string,
) ([]SubscriptionView, error) {
	subs, err := u.subscriptions.GetActiveByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	var result []SubscriptionView
	for _, sub := range subs {
		repository, err := u.repositories.GetRepository(ctx, sub.RepositoryID)
		if err != nil {
			return nil, err
		}
		result = append(result, SubscriptionView{
			Email:       email,
			Repo:        repository.Name,
			Confirmed:   sub.SubscriptionStatus == domain.StatusActive,
			LastSeenTag: repository.LastSeenTag,
		})
	}
	return result, nil
}
