package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
)

type SubscriberRepo interface {
	GetByEmail(ctx context.Context, email string) (*model.Subscriber, error)
	Create(ctx context.Context, email string) (*model.Subscriber, error)
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

func (s *SubscriberService) GetOrCreate(ctx context.Context, email string) (*model.Subscriber, error) {
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
