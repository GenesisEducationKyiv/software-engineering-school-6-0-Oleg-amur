package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/models"
	"github.com/google/uuid"
)

type subscriberService interface {
	GetOrCreate(ctx context.Context, email string) (*models.Subscriber, error)
}

type repositoryService interface {
	GetOrCreate(ctx context.Context, repoName string) (*models.Repository, error)
}

type SubscriptionRepo interface {
	Create(ctx context.Context, subID, repoID int, token string) error
	Activate(ctx context.Context, token string) error
	DeleteByToken(ctx context.Context, token string) error
	GetActiveByEmail(ctx context.Context, email string) ([]models.Subscription, error)
}

type Notifier interface {
	SendConfirmation(ctx context.Context, email, token string) error
	SendReleaseNotification(ctx context.Context, email, repo, tag string) error
}

type SubscriptionService struct {
	log               *slog.Logger
	subscriberService subscriberService
	repositoryService repositoryService
	subscriptionRepo  SubscriptionRepo
	notifier          Notifier
}

func NewSubscriptionService(
	log *slog.Logger,
	sub subscriberService,
	repo repositoryService,
	subscription SubscriptionRepo,
	notifier Notifier,
) *SubscriptionService {
	return &SubscriptionService{
		log:               log,
		subscriberService: sub,
		repositoryService: repo,
		subscriptionRepo:  subscription,
		notifier:          notifier,
	}
}

func (s *SubscriptionService) Subscribe(ctx context.Context, req models.SubscribeRequest) error {
	subscriber, err := s.subscriberService.GetOrCreate(ctx, req.Email)
	if err != nil {
		return err
	}

	repo, err := s.repositoryService.GetOrCreate(ctx, req.Repo)
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

	go func() {
		bgCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()

		if err := s.notifier.SendConfirmation(bgCtx, req.Email, token); err != nil {
			s.log.Error("failed to send notification", "err", err)
		}
	}()

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
) ([]models.SubscriptionDTO, error) {
	subs, err := s.subscriptionRepo.GetActiveByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	var result []models.SubscriptionDTO
	for _, sub := range subs {
		result = append(result, models.SubscriptionDTO{
			Email:       email,
			Repo:        sub.Repository.Name,
			Confirmed:   sub.SubscriptionStatus == models.StatusActive,
			LastSeenTag: sub.Repository.LastSeenTag,
		})
	}
	return result, nil
}
