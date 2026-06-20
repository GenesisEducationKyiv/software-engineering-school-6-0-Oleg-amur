package notification

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

func TestNotificationService_HandleReleaseNotificationRequested(t *testing.T) {
	tests := []struct {
		name      string
		event     events.ReleaseNotificationRequested
		senderErr error
		wantErr   error
	}{
		{
			name: "sends release email",
			event: events.ReleaseNotificationRequested{
				Email:            "user@example.com",
				Repo:             "owner/repo",
				Tag:              "v1.0.0",
				UnsubscribeToken: "unsubscribe-token",
			},
		},
		{
			name: "returns sender error",
			event: events.ReleaseNotificationRequested{
				Email:            "user@example.com",
				Repo:             "owner/repo",
				Tag:              "v1.0.0",
				UnsubscribeToken: "unsubscribe-token",
			},
			senderErr: errors.New("smtp error"),
			wantErr:   errors.New("smtp error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &mockEmailSender{err: tt.senderErr}
			builder := &mockMessageBuilder{}
			svc := NewNotificationService(testLogger(), sender, builder, nil)

			err := svc.HandleReleaseNotificationRequested(context.Background(), tt.event)

			if tt.wantErr == nil && err != nil {
				t.Fatalf("got error %v, want nil", err)
			}
			if tt.wantErr != nil && err == nil {
				t.Fatal("got nil error, want sender error")
			}
			assertSentEmailsLen(t, sender, 1)
			if builder.releaseRepo != tt.event.Repo {
				t.Errorf("got release repo %q, want %q", builder.releaseRepo, tt.event.Repo)
			}
			if builder.releaseTag != tt.event.Tag {
				t.Errorf("got release tag %q, want %q", builder.releaseTag, tt.event.Tag)
			}
			if builder.releaseToken != tt.event.UnsubscribeToken {
				t.Errorf("got release token %q, want %q", builder.releaseToken, tt.event.UnsubscribeToken)
			}
			if sender.sentEmails[0].to != tt.event.Email {
				t.Errorf("got email recipient %q, want %q", sender.sentEmails[0].to, tt.event.Email)
			}
			if sender.sentEmails[0].subject != "Release Subject" {
				t.Errorf("got release subject %q, want %q", sender.sentEmails[0].subject, "Release Subject")
			}
		})
	}
}

func TestNotificationService_HandleSubscriptionConfirmationRequested(t *testing.T) {
	sender := &mockEmailSender{}
	builder := &mockMessageBuilder{}
	publisher := &mockSubscriptionSagaResultPublisher{}
	svc := NewNotificationService(testLogger(), sender, builder, publisher)

	err := svc.HandleSubscriptionConfirmationRequested(context.Background(), events.SubscriptionConfirmationRequested{
		SagaID:            10,
		SubscriptionID:    20,
		Email:             "user@example.com",
		ConfirmationToken: "confirm-token",
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if builder.confirmationToken != "confirm-token" {
		t.Errorf("got confirmation token %q, want %q", builder.confirmationToken, "confirm-token")
	}
	assertSentEmailsLen(t, sender, 1)
	if sender.sentEmails[0].to != "user@example.com" {
		t.Errorf("got email recipient %q, want %q", sender.sentEmails[0].to, "user@example.com")
	}
	if sender.sentEmails[0].subject != "Confirm Subject" {
		t.Errorf("got confirmation subject %q, want %q", sender.sentEmails[0].subject, "Confirm Subject")
	}
	if len(publisher.succeeded) != 1 {
		t.Fatalf("got %d succeeded events, want 1", len(publisher.succeeded))
	}
	if publisher.succeeded[0].SagaID != 10 {
		t.Errorf("got saga id %d, want 10", publisher.succeeded[0].SagaID)
	}
}

func TestNotificationService_HandleSubscriptionConfirmationRequested_PublishesFailureResult(t *testing.T) {
	senderErr := errors.New("smtp error")
	sender := &mockEmailSender{err: senderErr}
	builder := &mockMessageBuilder{}
	publisher := &mockSubscriptionSagaResultPublisher{}
	svc := NewNotificationService(testLogger(), sender, builder, publisher)

	err := svc.HandleSubscriptionConfirmationRequested(context.Background(), events.SubscriptionConfirmationRequested{
		SagaID:            10,
		SubscriptionID:    20,
		Email:             "user@example.com",
		ConfirmationToken: "confirm-token",
	})

	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	assertSentEmailsLen(t, sender, 1)
	if len(publisher.failed) != 1 {
		t.Fatalf("got %d failed events, want 1", len(publisher.failed))
	}
	if publisher.failed[0].Reason != senderErr.Error() {
		t.Errorf("got failure reason %q, want %q", publisher.failed[0].Reason, senderErr.Error())
	}
}

func assertSentEmailsLen(t *testing.T, sender *mockEmailSender, want int) {
	t.Helper()

	if len(sender.sentEmails) != want {
		t.Fatalf("got %d sent emails, want %d", len(sender.sentEmails), want)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type sentEmail struct {
	to      string
	subject string
	body    string
}

type mockEmailSender struct {
	sentEmails []sentEmail
	err        error
}

func (f *mockEmailSender) Send(ctx context.Context, to, subject, body string) error {
	f.sentEmails = append(f.sentEmails, sentEmail{
		to:      to,
		subject: subject,
		body:    body,
	})
	return f.err
}

type mockMessageBuilder struct {
	confirmationToken string
	releaseRepo       string
	releaseTag        string
	releaseToken      string
}

func (f *mockMessageBuilder) BuildConfirmationMessage(token string) (string, string) {
	f.confirmationToken = token
	return "Confirm Subject", "Confirm Body"
}

func (f *mockMessageBuilder) BuildReleaseMessage(repo, tag, token string) (string, string) {
	f.releaseRepo = repo
	f.releaseTag = tag
	f.releaseToken = token
	return "Release Subject", "Release Body"
}

type mockSubscriptionSagaResultPublisher struct {
	succeeded []events.SubscriptionConfirmationSucceeded
	failed    []events.SubscriptionConfirmationFailed
}

func (f *mockSubscriptionSagaResultPublisher) PublishSubscriptionConfirmationSucceeded(
	ctx context.Context,
	event events.SubscriptionConfirmationSucceeded,
) error {
	f.succeeded = append(f.succeeded, event)
	return nil
}

func (f *mockSubscriptionSagaResultPublisher) PublishSubscriptionConfirmationFailed(
	ctx context.Context,
	event events.SubscriptionConfirmationFailed,
) error {
	f.failed = append(f.failed, event)
	return nil
}
