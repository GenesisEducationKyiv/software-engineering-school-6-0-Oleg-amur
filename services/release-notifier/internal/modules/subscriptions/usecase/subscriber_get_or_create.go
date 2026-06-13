package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

type SubscriberStore interface {
	GetByEmail(ctx context.Context, email string) (*domain.Subscriber, error)
	Create(ctx context.Context, email string) (*domain.Subscriber, error)
}

type GetOrCreateSubscriber struct {
	log             *slog.Logger
	subscriberStore SubscriberStore
}

func NewGetOrCreateSubscriber(
	log *slog.Logger,
	subscriberStore SubscriberStore,
) *GetOrCreateSubscriber {
	return &GetOrCreateSubscriber{
		log:             log,
		subscriberStore: subscriberStore,
	}
}

func (u *GetOrCreateSubscriber) Execute(ctx context.Context, email string) (*domain.Subscriber, error) {
	subscriber, err := u.subscriberStore.GetByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, apperr.ErrNotFound) {
			return nil, fmt.Errorf("subscriber check error: %w", err)
		}
		subscriber, err = u.subscriberStore.Create(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("subscriber create error: %w", err)
		}
	}

	return subscriber, nil
}
