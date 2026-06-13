package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/google/uuid"
)

type subscriptionRepoForReleaseNotifications interface {
	GetActiveRecipientsByRepoID(ctx context.Context, repoID int) ([]domain.NotificationRecipient, error)
}

type releaseNotificationPublisher interface {
	PublishReleaseNotification(ctx context.Context, event events.ReleaseNotificationRequested) error
}

type PlanReleaseNotifications struct {
	log              *slog.Logger
	subscriptionRepo subscriptionRepoForReleaseNotifications
	events           releaseNotificationPublisher
}

func NewPlanReleaseNotifications(
	log *slog.Logger,
	subscriptionRepo subscriptionRepoForReleaseNotifications,
	events releaseNotificationPublisher,
) *PlanReleaseNotifications {
	return &PlanReleaseNotifications{
		log:              log,
		subscriptionRepo: subscriptionRepo,
		events:           events,
	}
}

func (p *PlanReleaseNotifications) Execute(ctx context.Context, event domain.ReleaseEvent) error {
	recipients, err := p.subscriptionRepo.GetActiveRecipientsByRepoID(ctx, event.RepoID)
	if err != nil {
		return fmt.Errorf("fetch active subscriptions: %w", err)
	}

	for _, recipient := range recipients {
		notification := events.ReleaseNotificationRequested{
			EventID:          uuid.New().String(),
			Email:            recipient.Email,
			Repo:             event.RepoName,
			Tag:              event.Tag,
			UnsubscribeToken: recipient.UnsubscribeToken,
		}
		if err := p.events.PublishReleaseNotification(ctx, notification); err != nil {
			return fmt.Errorf("publish release notification event: %w", err)
		}
	}

	return nil
}
