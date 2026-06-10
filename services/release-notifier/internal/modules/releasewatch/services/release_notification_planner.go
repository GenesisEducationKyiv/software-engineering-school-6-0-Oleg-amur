package services

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasewatch/models"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/google/uuid"
)

type ReleaseNotificationPlanner struct {
	log              *slog.Logger
	subscriptionRepo subscriptionRepoForReleaseNotifications
	events           releaseNotificationPublisher
}

func NewReleaseNotificationPlanner(
	log *slog.Logger,
	subscriptionRepo subscriptionRepoForReleaseNotifications,
	events releaseNotificationPublisher,
) *ReleaseNotificationPlanner {
	return &ReleaseNotificationPlanner{
		log:              log,
		subscriptionRepo: subscriptionRepo,
		events:           events,
	}
}

func (p *ReleaseNotificationPlanner) HandleReleaseDetected(ctx context.Context, event models.ReleaseEvent) error {
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
