package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/contracts/events"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
	"github.com/google/uuid"
)

type subscriptionRepoForReleaseNotifications interface {
	GetActiveByRepoID(ctx context.Context, repoID int) ([]model.Subscription, error)
}

type releaseNotificationPublisher interface {
	PublishReleaseNotification(ctx context.Context, event events.ReleaseNotificationRequested) error
}

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

func (p *ReleaseNotificationPlanner) HandleReleaseDetected(ctx context.Context, event model.ReleaseEvent) error {
	subs, err := p.subscriptionRepo.GetActiveByRepoID(ctx, event.RepoID)
	if err != nil {
		return fmt.Errorf("fetch active subscriptions: %w", err)
	}

	for _, sub := range subs {
		if sub.Subscriber == nil {
			p.log.Error("active subscription has no subscriber", "subscription_id", sub.ID, "repo_id", event.RepoID)
			continue
		}

		notification := events.ReleaseNotificationRequested{
			EventID:          uuid.New().String(),
			Email:            sub.Subscriber.Email,
			Repo:             event.RepoName,
			Tag:              event.Tag,
			UnsubscribeToken: sub.Token,
		}
		if err := p.events.PublishReleaseNotification(ctx, notification); err != nil {
			return fmt.Errorf("publish release notification event: %w", err)
		}
	}

	return nil
}
