package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/http/dto"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/models"
	"github.com/google/uuid"
)

type SubscriberRepo interface {
	GetByEmail(ctx context.Context, email string) (*models.Subscriber, error)
	Create(ctx context.Context, email string) (*models.Subscriber, error)
}

type RepositoryRepo interface {
	GetByName(ctx context.Context, name string) (*models.Repository, error)
	Create(ctx context.Context, name string, lastSeenTag string) (*models.Repository, error)
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

type GithubClient interface {
	GetRepositoryLatestTag(ctx context.Context, repoAddr string, log *slog.Logger) (string, error)
	CheckIfRepoExists(ctx context.Context, repoAddr string, log *slog.Logger) (bool, error)
}

type SubscriptionService struct {
	log              *slog.Logger
	subscriberRepo   SubscriberRepo
	repositoryRepo   RepositoryRepo
	subscriptionRepo SubscriptionRepo
	notifier         Notifier
	githubClient     GithubClient
}

func NewSubscriptionService(
	log *slog.Logger,
	sub SubscriberRepo,
	repo RepositoryRepo,
	subscription SubscriptionRepo,
	notifier Notifier,
	githubClient GithubClient,
) *SubscriptionService {
	return &SubscriptionService{
		log:              log,
		subscriberRepo:   sub,
		repositoryRepo:   repo,
		subscriptionRepo: subscription,
		notifier:         notifier,
		githubClient:     githubClient,
	}
}

func (s *SubscriptionService) Subscribe(ctx context.Context, req dto.SubscribeRequest) error {
	parts := strings.Split(req.Repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return apperr.ErrInvalidFormat
	}

	subscriber, err := s.subscriberRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("subscriber check error: %w", err)
		}
		subscriber, err = s.subscriberRepo.Create(ctx, req.Email)
		if err != nil {
			return fmt.Errorf("subscriber create error: %w", err)
		}
	}

	repo, err := s.repositoryRepo.GetByName(ctx, req.Repo)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("repository check error: %w", err)
		}

		exists, checkErr := s.githubClient.CheckIfRepoExists(ctx, req.Repo, s.log)
		if checkErr != nil {
			if errors.Is(checkErr, apperr.ErrRateLimitExceeded) {
				return apperr.ErrRateLimitExceeded
			}
			return fmt.Errorf("github check existence failed: %w", checkErr)
		}
		if !exists {
			return apperr.ErrRepoNotFound
		}

		tag, tagErr := s.githubClient.GetRepositoryLatestTag(ctx, req.Repo, s.log)
		if tagErr != nil {
			if errors.Is(tagErr, apperr.ErrRateLimitExceeded) {
				return apperr.ErrRateLimitExceeded
			}
			if !errors.Is(tagErr, apperr.ErrRepoNotFound) {
				return fmt.Errorf("github get tag failed: %w", tagErr)
			}
		}

		repo, err = s.repositoryRepo.Create(ctx, req.Repo, tag)
		if err != nil {
			return fmt.Errorf("failed to create repository: %w", err)
		}
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
		if errors.Is(err, sql.ErrNoRows) {
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
) ([]dto.Subscription, error) {
	subs, err := s.subscriptionRepo.GetActiveByEmail(ctx, email)
	if err != nil {
		return nil, err
	}

	var result []dto.Subscription
	for _, sub := range subs {
		result = append(result, dto.Subscription{
			Email:       email,
			Repo:        sub.Repository.Name,
			Confirmed:   sub.SubscriptionStatus == models.StatusActive,
			LastSeenTag: sub.Repository.LastSeenTag,
		})
	}
	return result, nil
}
