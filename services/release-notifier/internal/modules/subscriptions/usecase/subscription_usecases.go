package usecase

import (
	"context"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

type SubscriptionUsecases struct {
	SubscribeToRepository     *SubscribeToRepository
	ConfirmSubscription       *ConfirmSubscription
	UnsubscribeFromRepository *UnsubscribeFromRepository
	ListSubscriptions         *ListSubscriptions
	ListActiveByRepository    *ListActiveSubscriptionsByRepository
}

func (u SubscriptionUsecases) GetActiveSubscriptionsByRepository(
	ctx context.Context,
	repoName string,
) ([]domain.RepositorySubscription, error) {
	return u.ListActiveByRepository.Execute(ctx, repoName)
}

func (u SubscriptionUsecases) Subscribe(ctx context.Context, req SubscribeRequest) error {
	return u.SubscribeToRepository.Execute(ctx, req)
}

func (u SubscriptionUsecases) Confirm(ctx context.Context, token string) error {
	return u.ConfirmSubscription.Execute(ctx, token)
}

func (u SubscriptionUsecases) Unsubscribe(ctx context.Context, token string) error {
	return u.UnsubscribeFromRepository.Execute(ctx, token)
}

func (u SubscriptionUsecases) GetSubscriptions(
	ctx context.Context,
	email string,
) ([]SubscriptionView, error) {
	return u.ListSubscriptions.Execute(ctx, email)
}
