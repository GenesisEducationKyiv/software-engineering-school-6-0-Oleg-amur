package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
	"github.com/google/uuid"
)

type subscriberService interface {
	GetOrCreate(ctx context.Context, email string) (*model.Subscriber, error)
}

type repositoryService interface {
	GetOrCreate(ctx context.Context, repoName string) (*model.Repository, error)
}

type SubscriptionRepo interface {
	Create(ctx context.Context, subID, repoID int, token string) error
	Activate(ctx context.Context, token string) error
	DeleteByToken(ctx context.Context, token string) error
	GetActiveByEmail(ctx context.Context, email string) ([]model.Subscription, error)
}

type SubscriptionService struct {
	log               *slog.Logger
	subscriberService subscriberService
	repositoryService repositoryService
	subscriptionRepo  SubscriptionRepo
	subsChan          chan<- model.SubscriptionEvent
}

func NewSubscriptionService(
	log *slog.Logger,
	sub subscriberService,
	repo repositoryService,
	subscription SubscriptionRepo,
	subsChan chan<- model.SubscriptionEvent,
) *SubscriptionService {
	return &SubscriptionService{
		log:               log,
		subscriberService: sub,
		repositoryService: repo,
		subscriptionRepo:  subscription,
		subsChan:          subsChan,
	}
}

func (s *SubscriptionService) Subscribe(ctx context.Context, req model.SubscribeRequest) error {
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

	select {
	case s.subsChan <- model.SubscriptionEvent{Email: req.Email, Token: token}:
	default:
		s.log.Error("failed to enqueue subscription event, channel full", "email", req.Email)
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
) ([]model.SubscriptionDTO, error) {
	subs, err := s.subscriptionRepo.GetActiveByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	var result []model.SubscriptionDTO
	for _, sub := range subs {
		result = append(result, model.SubscriptionDTO{
			Email:       email,
			Repo:        sub.Repository.Name,
			Confirmed:   sub.SubscriptionStatus == model.StatusActive,
			LastSeenTag: sub.Repository.LastSeenTag,
		})
	}
	return result, nil
}
