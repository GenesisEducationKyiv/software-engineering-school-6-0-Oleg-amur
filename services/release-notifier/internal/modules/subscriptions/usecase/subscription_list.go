package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

type ActiveSubscriptionStore interface {
	GetActiveByEmail(ctx context.Context, email string) ([]domain.Subscription, error)
}

type ListSubscriptions struct {
	subscriptions ActiveSubscriptionStore
}

func NewListSubscriptions(subscriptions ActiveSubscriptionStore) *ListSubscriptions {
	return &ListSubscriptions{subscriptions: subscriptions}
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
		result = append(result, SubscriptionView{
			Email:       email,
			Repo:        sub.Repository.Name,
			Confirmed:   sub.SubscriptionStatus == domain.StatusActive,
			LastSeenTag: sub.Repository.LastSeenTag,
		})
	}
	return result, nil
}
