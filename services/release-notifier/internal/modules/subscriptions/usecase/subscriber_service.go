package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

type SubscriberRepo interface {
	GetByEmail(ctx context.Context, email string) (*domain.Subscriber, error)
	Create(ctx context.Context, email string) (*domain.Subscriber, error)
}

type SubscriberService struct {
	log            *slog.Logger
	subscriberRepo SubscriberRepo
}

func NewSubscriberService(
	log *slog.Logger,
	sub SubscriberRepo,
) *SubscriberService {
	return &SubscriberService{
		log:            log,
		subscriberRepo: sub,
	}
}

func (s *SubscriberService) GetOrCreate(ctx context.Context, email string) (*domain.Subscriber, error) {
	subscriber, err := s.subscriberRepo.GetByEmail(ctx, email)
	if err != nil {
		if !errors.Is(err, apperr.ErrNotFound) {
			return nil, fmt.Errorf("subscriber check error: %w", err)
		}
		subscriber, err = s.subscriberRepo.Create(ctx, email)
		if err != nil {
			return nil, fmt.Errorf("subscriber create error: %w", err)
		}
	}

	return subscriber, nil
}
