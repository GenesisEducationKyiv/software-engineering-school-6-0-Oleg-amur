package notification

import (
	"context"
	"errors"
	"fmt"
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

type SubscriptionSagaResultPublisher interface {
	PublishSubscriptionConfirmationSucceeded(ctx context.Context, event events.SubscriptionConfirmationSucceeded) error
	PublishSubscriptionConfirmationFailed(ctx context.Context, event events.SubscriptionConfirmationFailed) error
}

type NotificationService struct {
	log            *slog.Logger
	emailSender    EmailSender
	messageBuilder MessageBuilder
	results        SubscriptionSagaResultPublisher
}

func NewNotificationService(
	log *slog.Logger,
	emailSender EmailSender,
	messageBuilder MessageBuilder,
	results SubscriptionSagaResultPublisher,
) *NotificationService {
	return &NotificationService{
		log:            log,
		emailSender:    emailSender,
		messageBuilder: messageBuilder,
		results:        results,
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

	subject, body := s.messageBuilder.BuildConfirmationMessage(event.ConfirmationToken)

	if err := s.emailSender.Send(ctx, event.Email, subject, body); err != nil {
		s.log.Error("failed to send confirmation", "email", event.Email, "err", err)
		if s.results == nil {
			return err
		}
		if publishErr := s.publishSubscriptionConfirmationFailed(ctx, event, err); publishErr != nil {
			return errors.Join(err, publishErr)
		}
		return nil
	}

	return s.publishSubscriptionConfirmationSucceeded(ctx, event)
}

func (s *NotificationService) publishSubscriptionConfirmationSucceeded(
	ctx context.Context,
	request events.SubscriptionConfirmationRequested,
) error {
	if s.results == nil {
		return nil
	}

	event := events.SubscriptionConfirmationSucceeded{
		EventID:           fmt.Sprintf("%s.succeeded", request.EventID),
		SchemaVersion:     events.NotificationSchemaVersion,
		SagaID:            request.SagaID,
		SubscriptionID:    request.SubscriptionID,
		Email:             request.Email,
		ConfirmationToken: request.ConfirmationToken,
	}
	if err := s.results.PublishSubscriptionConfirmationSucceeded(ctx, event); err != nil {
		s.log.Error("failed to publish subscription confirmation succeeded event", "email", request.Email, "err", err)
		return err
	}
	return nil
}

func (s *NotificationService) publishSubscriptionConfirmationFailed(
	ctx context.Context,
	request events.SubscriptionConfirmationRequested,
	cause error,
) error {
	if s.results == nil {
		return nil
	}

	event := events.SubscriptionConfirmationFailed{
		EventID:           fmt.Sprintf("%s.failed", request.EventID),
		SchemaVersion:     events.NotificationSchemaVersion,
		SagaID:            request.SagaID,
		SubscriptionID:    request.SubscriptionID,
		Email:             request.Email,
		ConfirmationToken: request.ConfirmationToken,
		Reason:            cause.Error(),
	}
	if err := s.results.PublishSubscriptionConfirmationFailed(ctx, event); err != nil {
		s.log.Error("failed to publish subscription confirmation failed event", "email", request.Email, "err", err)
		return err
	}
	return nil
}
