package usecase

import "context"

type SubscriptionDeleter interface {
	DeleteByToken(ctx context.Context, token string) error
}

type UnsubscribeFromRepository struct {
	subscriptions SubscriptionDeleter
}

func NewUnsubscribeFromRepository(subscriptions SubscriptionDeleter) *UnsubscribeFromRepository {
	return &UnsubscribeFromRepository{subscriptions: subscriptions}
}

func (u *UnsubscribeFromRepository) Execute(ctx context.Context, token string) error {
	return u.subscriptions.DeleteByToken(ctx, token)
}
