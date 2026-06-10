package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
)

func TestNotificationService_ProcessReleaseEvent(t *testing.T) {
	tests := []struct {
		name       string
		event      model.ReleaseEvent
		subs       []model.Subscription
		repoErr    error
		senderErr  error
		wantEmails int
	}{
		{
			name: "sends release email to each active subscriber",
			event: model.ReleaseEvent{
				RepoID:   1,
				RepoName: "owner/repo",
				Tag:      "v1.0.0",
			},
			subs: []model.Subscription{
				{Subscriber: &model.Subscriber{Email: "user1@example.com"}, Token: "token-1"},
				{Subscriber: &model.Subscriber{Email: "user2@example.com"}, Token: "token-2"},
			},
			wantEmails: 2,
		},
		{
			name: "skips sending when repository lookup fails",
			event: model.ReleaseEvent{
				RepoID:   1,
				RepoName: "owner/repo",
				Tag:      "v1.0.0",
			},
			repoErr: errors.New("db error"),
		},
		{
			name: "attempts all emails when sender returns errors",
			event: model.ReleaseEvent{
				RepoID:   1,
				RepoName: "owner/repo",
				Tag:      "v1.0.0",
			},
			subs: []model.Subscription{
				{Subscriber: &model.Subscriber{Email: "user1@example.com"}, Token: "token-1"},
				{Subscriber: &model.Subscriber{Email: "user2@example.com"}, Token: "token-2"},
			},
			senderErr:  errors.New("smtp error"),
			wantEmails: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockNotificationSubRepo{
				subs: tt.subs,
				err:  tt.repoErr,
			}
			sender := &mockEmailSender{
				err: tt.senderErr,
			}
			builder := &mockMessageBuilder{}

			svc := NewNotificationService(testLogger(), repo, sender, builder)
			svc.processReleaseEvent(context.Background(), tt.event)

			assertSentEmailsLen(t, sender, tt.wantEmails)
		})
	}
}

func TestNotificationService_ProcessReleaseEvent_BuildsReleaseMessage(t *testing.T) {
	repo := &mockNotificationSubRepo{
		subs: []model.Subscription{
			{Subscriber: &model.Subscriber{Email: "user@example.com"}, Token: "unsubscribe-token"},
		},
	}
	sender := &mockEmailSender{}
	builder := &mockMessageBuilder{}
	svc := NewNotificationService(testLogger(), repo, sender, builder)

	svc.processReleaseEvent(context.Background(), model.ReleaseEvent{
		RepoID:   1,
		RepoName: "owner/repo",
		Tag:      "v1.0.0",
	})

	if builder.releaseRepo != "owner/repo" {
		t.Errorf("got release repo %q, want %q", builder.releaseRepo, "owner/repo")
	}
	if builder.releaseTag != "v1.0.0" {
		t.Errorf("got release tag %q, want %q", builder.releaseTag, "v1.0.0")
	}
	if builder.releaseToken != "unsubscribe-token" {
		t.Errorf("got release token %q, want %q", builder.releaseToken, "unsubscribe-token")
	}
	assertSentEmailsLen(t, sender, 1)
	if sender.sentEmails[0].to != "user@example.com" {
		t.Errorf("got email recipient %q, want %q", sender.sentEmails[0].to, "user@example.com")
	}
	if sender.sentEmails[0].subject != "Release Subject" {
		t.Errorf("got release subject %q, want %q", sender.sentEmails[0].subject, "Release Subject")
	}
}

func TestNotificationService_ProcessSubscriptionEvent(t *testing.T) {
	repo := &mockNotificationSubRepo{}
	sender := &mockEmailSender{}
	builder := &mockMessageBuilder{}
	svc := NewNotificationService(testLogger(), repo, sender, builder)

	svc.processSubscriptionEvent(context.Background(), model.SubscriptionEvent{
		Email: "user@example.com",
		Token: "confirm-token",
	})

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
}

func TestNotificationService_StartAndStop(t *testing.T) {
	repo := &mockNotificationSubRepo{
		subs: []model.Subscription{
			{Subscriber: &model.Subscriber{Email: "user@example.com"}, Token: "token"},
		},
	}
	sender := &mockEmailSender{}
	builder := &mockMessageBuilder{}
	svc := NewNotificationService(testLogger(), repo, sender, builder)

	eventsChan := make(chan model.ReleaseEvent)
	subsChan := make(chan model.SubscriptionEvent)
	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		defer close(done)
		svc.Start(ctx, eventsChan, subsChan)
	}()

	eventsChan <- model.ReleaseEvent{
		RepoID:   1,
		RepoName: "owner/test",
		Tag:      "v1",
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("notification service did not stop")
	}
}

func assertSentEmailsLen(t *testing.T, sender *mockEmailSender, want int) {
	t.Helper()

	if len(sender.sentEmails) != want {
		t.Fatalf("got %d sent emails, want %d", len(sender.sentEmails), want)
	}
}

type sentEmail struct {
	to      string
	subject string
	body    string
}

type mockNotificationSubRepo struct {
	subs []model.Subscription
	err  error
}

func (f *mockNotificationSubRepo) GetActiveByRepoID(ctx context.Context, repoID int) ([]model.Subscription, error) {
	return f.subs, f.err
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
