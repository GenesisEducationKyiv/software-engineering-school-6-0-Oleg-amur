package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	releasetrackerdomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	"github.com/google/uuid"
)

type SubscriberRegistration interface {
	Execute(ctx context.Context, email string) (*domain.Subscriber, error)
}

type RepositoryTracker interface {
	Execute(ctx context.Context, repoName string) (*releasetrackerdomain.Repository, error)
}

type SubscriptionCreator interface {
	StartSubscriptionConfirmation(ctx context.Context, subID, repoID int, email string, token string) error
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

	repo, err := u.repositoryTracker.Execute(ctx, req.Repo)
	if err != nil {
		return err
	}

	token := uuid.New().String()
	err = u.subscriptions.StartSubscriptionConfirmation(ctx, subscriber.ID, repo.ID, req.Email, token)
	if err != nil {
		if errors.Is(err, apperr.ErrAlreadyExists) {
			return apperr.ErrAlreadySubscribed
		}
		return fmt.Errorf("subscription error: %w", err)
	}

	return nil
}
