package notification

import (
	"context"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

type EmailSender interface {
	Send(ctx context.Context, to, subject, body string) error
}

type MessageBuilder interface {
	BuildConfirmationMessage(token string) (subject, body string)
	BuildReleaseMessage(repo, tag, token string) (subject, body string)
}

type NotificationService struct {
	log            *slog.Logger
	emailSender    EmailSender
	messageBuilder MessageBuilder
}

func NewNotificationService(
	log *slog.Logger,
	emailSender EmailSender,
	messageBuilder MessageBuilder,
) *NotificationService {
	return &NotificationService{
		log:            log,
		emailSender:    emailSender,
		messageBuilder: messageBuilder,
	}
}

func (s *NotificationService) HandleReleaseNotificationRequested(
	ctx context.Context,
	event events.ReleaseNotificationRequested,
) error {
	s.log.Info("sending release notification", "email", event.Email, "repo", event.Repo, "tag", event.Tag)

	subject, body := s.messageBuilder.BuildReleaseMessage(event.Repo, event.Tag, event.UnsubscribeToken)

	if err := s.emailSender.Send(ctx, event.Email, subject, body); err != nil {
		s.log.Error("failed to send release notification", "email", event.Email, "err", err)
		return err
	}

	return nil
}

func (s *NotificationService) HandleSubscriptionConfirmationRequested(
	ctx context.Context,
	event events.SubscriptionConfirmationRequested,
) error {
	s.log.Info("sending confirmation email", "email", event.Email)

	subject, body := s.messageBuilder.BuildConfirmationMessage(event.Token)

	if err := s.emailSender.Send(ctx, event.Email, subject, body); err != nil {
		s.log.Error("failed to send confirmation", "email", event.Email, "err", err)
		return err
	}

	return nil
}
