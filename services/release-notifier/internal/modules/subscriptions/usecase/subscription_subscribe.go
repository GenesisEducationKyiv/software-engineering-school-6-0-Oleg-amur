package usecase

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	releasetrackerdomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/google/uuid"
)

type SubscriberRegistration interface {
	Execute(ctx context.Context, email string) (*domain.Subscriber, error)
}

type RepositoryTracker interface {
	Execute(ctx context.Context, repoName string) (*releasetrackerdomain.Repository, error)
}

type SubscriptionCreator interface {
	Create(ctx context.Context, subID, repoID int, token string) error
}

type ConfirmationPublisher interface {
	PublishSubscriptionConfirmation(ctx context.Context, event events.SubscriptionConfirmationRequested) error
}

type SubscribeToRepository struct {
	log                    *slog.Logger
	subscriberRegistration SubscriberRegistration
	repositoryTracker      RepositoryTracker
	subscriptions          SubscriptionCreator
	events                 ConfirmationPublisher
}

func NewSubscribeToRepository(
	log *slog.Logger,
	subscriberRegistration SubscriberRegistration,
	repositoryTracker RepositoryTracker,
	subscriptions SubscriptionCreator,
	events ConfirmationPublisher,
) *SubscribeToRepository {
	return &SubscribeToRepository{
		log:                    log,
		subscriberRegistration: subscriberRegistration,
		repositoryTracker:      repositoryTracker,
		subscriptions:          subscriptions,
		events:                 events,
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
	err = u.subscriptions.Create(ctx, subscriber.ID, repo.ID, token)
	if err != nil {
		if errors.Is(err, apperr.ErrAlreadyExists) {
			return apperr.ErrAlreadySubscribed
		}
		return fmt.Errorf("subscription error: %w", err)
	}

	event := events.SubscriptionConfirmationRequested{
		EventID:           uuid.New().String(),
		SchemaVersion:     events.NotificationSchemaVersion,
		Email:             req.Email,
		ConfirmationToken: token,
	}
	if err := u.events.PublishSubscriptionConfirmation(ctx, event); err != nil {
		u.log.Error("failed to publish subscription confirmation event", "email", req.Email, "err", err)
		return fmt.Errorf("publish subscription confirmation event: %w", err)
	}

	return nil
}
