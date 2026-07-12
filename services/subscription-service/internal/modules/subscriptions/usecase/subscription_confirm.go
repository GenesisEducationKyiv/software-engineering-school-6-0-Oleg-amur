package usecase

import (
	"context"
	"errors"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/apperr"
)

type SubscriptionActivator interface {
	Activate(ctx context.Context, token string) error
}

type ConfirmSubscription struct {
	subscriptions SubscriptionActivator
}

func NewConfirmSubscription(subscriptions SubscriptionActivator) *ConfirmSubscription {
	return &ConfirmSubscription{subscriptions: subscriptions}
}

func (u *ConfirmSubscription) Execute(ctx context.Context, token string) error {
	err := u.subscriptions.Activate(ctx, token)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return apperr.ErrTokenNotFound
		}
		return err
	}
	return nil
}
