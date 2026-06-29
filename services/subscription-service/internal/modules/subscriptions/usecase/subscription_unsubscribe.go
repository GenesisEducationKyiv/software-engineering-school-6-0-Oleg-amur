package usecase

import (
	"context"
	"errors"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/apperr"
)

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
	err := u.subscriptions.DeleteByToken(ctx, token)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return apperr.ErrTokenNotFound
		}
		return err
	}
	return nil
}
