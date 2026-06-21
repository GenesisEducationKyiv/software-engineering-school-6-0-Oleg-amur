package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	"github.com/google/uuid"
)

type SubscriberRegistration interface {
	Execute(ctx context.Context, email string) (*domain.Subscriber, error)
}

type RepositoryTracker interface {
	EnsureTracked(ctx context.Context, repoName string) (*RepositoryView, error)
}

type SubscriptionCreator interface {
	StartSubscriptionConfirmation(ctx context.Context, subID int, repoName, email, token string) error
}

type SubscribeToRepository struct {
	log                    *slog.Logger
	subscriberRegistration SubscriberRegistration
	repositoryTracker      RepositoryTracker
	subscriptions          SubscriptionCreator
}

func NewSubscribeToRepository(
	log *slog.Logger,
	subscriberRegistration SubscriberRegistration,
	repositoryTracker RepositoryTracker,
	subscriptions SubscriptionCreator,
) *SubscribeToRepository {
	return &SubscribeToRepository{
		log:                    log,
		subscriberRegistration: subscriberRegistration,
		repositoryTracker:      repositoryTracker,
		subscriptions:          subscriptions,
	}
}

func (u *SubscribeToRepository) Execute(ctx context.Context, req SubscribeRequest) error {
	subscriber, err := u.subscriberRegistration.Execute(ctx, req.Email)
	if err != nil {
		return err
	}

	repo, err := u.repositoryTracker.EnsureTracked(ctx, req.Repo)
	if err != nil {
		return err
	}

	token := uuid.New().String()
	err = u.subscriptions.StartSubscriptionConfirmation(ctx, subscriber.ID, repo.Name, req.Email, token)
	if err != nil {
		if errors.Is(err, apperr.ErrAlreadyExists) {
			return apperr.ErrAlreadySubscribed
		}
		return fmt.Errorf("subscription error: %w", err)
	}

	return nil
}
