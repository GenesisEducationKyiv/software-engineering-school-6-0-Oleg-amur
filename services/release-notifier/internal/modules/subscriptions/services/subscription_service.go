package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	subscriptiondto "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/dto"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/models"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/google/uuid"
)

type SubscriptionService struct {
	log               *slog.Logger
	subscriberService subscriberService
	repositoryTracker RepositoryTracker
	subscriptionRepo  SubscriptionRepo
	events            notificationPublisher
}

func NewSubscriptionService(
	log *slog.Logger,
	sub subscriberService,
	repo RepositoryTracker,
	subscription SubscriptionRepo,
	events notificationPublisher,
) *SubscriptionService {
	return &SubscriptionService{
		log:               log,
		subscriberService: sub,
		repositoryTracker: repo,
		subscriptionRepo:  subscription,
		events:            events,
	}
}

func (s *SubscriptionService) Subscribe(ctx context.Context, req subscriptiondto.SubscribeRequest) error {
	subscriber, err := s.subscriberService.GetOrCreate(ctx, req.Email)
	if err != nil {
		return err
	}

	repo, err := s.repositoryTracker.EnsureTracked(ctx, req.Repo)
	if err != nil {
		return err
	}

	token := uuid.New().String()
	err = s.subscriptionRepo.Create(ctx, subscriber.ID, repo.ID, token)
	if err != nil {
		if errors.Is(err, apperr.ErrAlreadyExists) {
			return apperr.ErrAlreadySubscribed
		}
		return fmt.Errorf("subscription error: %w", err)
	}

	event := events.SubscriptionConfirmationRequested{
		EventID: uuid.New().String(),
		Email:   req.Email,
		Token:   token,
	}
	if err := s.events.PublishSubscriptionConfirmation(ctx, event); err != nil {
		s.log.Error("failed to publish subscription confirmation event", "email", req.Email, "err", err)
		return fmt.Errorf("publish subscription confirmation event: %w", err)
	}

	return nil
}

func (s *SubscriptionService) Confirm(ctx context.Context, token string) error {
	err := s.subscriptionRepo.Activate(ctx, token)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return apperr.ErrTokenNotFound
		}
		return err
	}
	return nil
}

func (s *SubscriptionService) Unsubscribe(ctx context.Context, token string) error {
	return s.subscriptionRepo.DeleteByToken(ctx, token)
}

func (s *SubscriptionService) GetSubscriptions(
	ctx context.Context,
	email string,
) ([]subscriptiondto.SubscriptionDTO, error) {
	subs, err := s.subscriptionRepo.GetActiveByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	var result []subscriptiondto.SubscriptionDTO
	for _, sub := range subs {
		result = append(result, subscriptiondto.SubscriptionDTO{
			Email:       email,
			Repo:        sub.Repository.Name,
			Confirmed:   sub.SubscriptionStatus == models.StatusActive,
			LastSeenTag: sub.Repository.LastSeenTag,
		})
	}
	return result, nil
}
