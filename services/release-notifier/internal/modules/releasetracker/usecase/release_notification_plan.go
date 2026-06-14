package usecase

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/domain"
	subscriptiondomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/google/uuid"
)

type ActiveSubscriptionsByRepositoryLister interface {
	Execute(ctx context.Context, repoID int) ([]subscriptiondomain.RepositorySubscription, error)
}

type releaseNotificationPublisher interface {
	PublishReleaseNotification(ctx context.Context, event events.ReleaseNotificationRequested) error
}

type PlanReleaseNotifications struct {
	log           *slog.Logger
	subscriptions ActiveSubscriptionsByRepositoryLister
	events        releaseNotificationPublisher
}

func NewPlanReleaseNotifications(
	log *slog.Logger,
	subscriptions ActiveSubscriptionsByRepositoryLister,
	events releaseNotificationPublisher,
) *PlanReleaseNotifications {
	return &PlanReleaseNotifications{
		log:           log,
		subscriptions: subscriptions,
		events:        events,
	}
}

func (p *PlanReleaseNotifications) Execute(ctx context.Context, event domain.ReleaseEvent) error {
	subscriptions, err := p.subscriptions.Execute(ctx, event.RepoID)
	if err != nil {
		return fmt.Errorf("fetch active subscriptions: %w", err)
	}

	for _, subscription := range subscriptions {
		notification := events.ReleaseNotificationRequested{
			EventID:          uuid.New().String(),
			SchemaVersion:    events.NotificationSchemaVersion,
			Email:            subscription.Email,
			Repo:             event.RepoName,
			Tag:              event.Tag,
			UnsubscribeToken: subscription.UnsubscribeToken,
		}
		if err := p.events.PublishReleaseNotification(ctx, notification); err != nil {
			return fmt.Errorf("publish release notification event: %w", err)
		}
	}

	return nil
}
