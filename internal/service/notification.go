package service

import (
	"context"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/models"
)

type subscriptionRepoForNotifications interface {
	GetActiveByRepoID(ctx context.Context, repoID int) ([]models.Subscription, error)
}

type Notifier interface {
	SendReleaseNotification(ctx context.Context, email, repo, tag string) error
	SendConfirmation(ctx context.Context, email, token string) error
}

type NotificationService struct {
	log              *slog.Logger
	subscriptionRepo subscriptionRepoForNotifications
	notifier         Notifier
}

func NewNotificationService(
	log *slog.Logger,
	subscriptionRepo subscriptionRepoForNotifications,
	notifier Notifier,
) *NotificationService {
	return &NotificationService{
		log:              log,
		subscriptionRepo: subscriptionRepo,
		notifier:         notifier,
	}
}

func (s *NotificationService) Start(
	ctx context.Context,
	releaseEvents <-chan models.ReleaseEvent,
	subscriptionEvents <-chan models.SubscriptionEvent,
) {
	s.log.Info("background notification service started")

	for {
		select {
		case <-ctx.Done():
			s.log.Info("background notification service stopping")
			return
		case event := <-releaseEvents:
			s.processReleaseEvent(ctx, event)
		case event := <-subscriptionEvents:
			s.processSubscriptionEvent(ctx, event)
		}
	}
}

func (s *NotificationService) processReleaseEvent(ctx context.Context, event models.ReleaseEvent) {
	subs, err := s.subscriptionRepo.GetActiveByRepoID(ctx, event.RepoID)
	if err != nil {
		s.log.Error("failed to fetch subscribers for notification", "repo", event.RepoName, "err", err)
		return
	}

	for _, sub := range subs {
		s.log.Info("sending notification", "email", sub.Subscriber.Email, "repo", event.RepoName, "tag", event.Tag)
		if err := s.notifier.SendReleaseNotification(ctx, sub.Subscriber.Email, event.RepoName, event.Tag); err != nil {
			s.log.Error("failed to send notification", "email", sub.Subscriber.Email, "err", err)
		}
	}
}

func (s *NotificationService) processSubscriptionEvent(ctx context.Context, event models.SubscriptionEvent) {
	s.log.Info("sending confirmation email", "email", event.Email)
	if err := s.notifier.SendConfirmation(ctx, event.Email, event.Token); err != nil {
		s.log.Error("failed to send confirmation", "email", event.Email, "err", err)
	}
}
