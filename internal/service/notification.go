package service

import (
	"context"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/models"
)

type subscriptionRepoForNotifications interface {
	GetActiveByRepoID(ctx context.Context, repoID int) ([]models.Subscription, error)
}

type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

type MessageBuilder interface {
	BuildConfirmationMessage(token string) (subject, body string)
	BuildReleaseMessage(repo, tag, token string) (subject, body string)
}

type NotificationService struct {
	log              *slog.Logger
	subscriptionRepo subscriptionRepoForNotifications
	emailSender      EmailSender
	messageBuilder   MessageBuilder
}

func NewNotificationService(
	log *slog.Logger,
	subscriptionRepo subscriptionRepoForNotifications,
	emailSender EmailSender,
	messageBuilder MessageBuilder,
) *NotificationService {
	return &NotificationService{
		log:              log,
		subscriptionRepo: subscriptionRepo,
		emailSender:      emailSender,
		messageBuilder:   messageBuilder,
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

		subject, body := s.messageBuilder.BuildReleaseMessage(event.RepoName, event.Tag, sub.Token)

		if err := s.emailSender.Send(ctx, sub.Subscriber.Email, subject, body); err != nil {
			s.log.Error("failed to send notification", "email", sub.Subscriber.Email, "err", err)
		}
	}
}

func (s *NotificationService) processSubscriptionEvent(ctx context.Context, event models.SubscriptionEvent) {
	s.log.Info("sending confirmation email", "email", event.Email)

	subject, body := s.messageBuilder.BuildConfirmationMessage(event.Token)

	if err := s.emailSender.Send(ctx, event.Email, subject, body); err != nil {
		s.log.Error("failed to send confirmation", "email", event.Email, "err", err)
	}
}
