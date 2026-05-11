package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/models"
)

type mockNotificationSubRepo struct {
	subs []models.Subscription
	err  error
}

func (m *mockNotificationSubRepo) GetActiveByRepoID(ctx context.Context, repoID int) ([]models.Subscription, error) {
	return m.subs, m.err
}

type mockEmailSender struct {
	sentEmails []string
	err        error
}

func (m *mockEmailSender) Send(ctx context.Context, to, subject, body string) error {
	m.sentEmails = append(m.sentEmails, to)
	return m.err
}

type mockMessageBuilder struct{}

func (m *mockMessageBuilder) BuildConfirmationMessage(token string) (string, string) {
	return "Confirm Subject", "Confirm Body"
}

func (m *mockMessageBuilder) BuildReleaseMessage(repo, tag, token string) (string, string) {
	return "Release Subject", "Release Body"
}

func TestNotificationService_processEvent(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tests := []struct {
		name           string
		event          models.ReleaseEvent
		mockSubs       []models.Subscription
		repoErr        error
		senderErr      error
		expectedEmails int
	}{
		{
			name: "Success - sends to multiple subscribers",
			event: models.ReleaseEvent{
				RepoID:   1,
				RepoName: "owner/repo",
				Tag:      "v1.0.0",
			},
			mockSubs: []models.Subscription{
				{Subscriber: &models.Subscriber{Email: "user1@example.com"}},
				{Subscriber: &models.Subscriber{Email: "user2@example.com"}},
			},
			repoErr:        nil,
			senderErr:      nil,
			expectedEmails: 2,
		},
		{
			name: "Repository error - skips sending",
			event: models.ReleaseEvent{
				RepoID:   1,
				RepoName: "owner/repo",
				Tag:      "v1.0.0",
			},
			mockSubs:       nil,
			repoErr:        errors.New("db error"),
			senderErr:      nil,
			expectedEmails: 0,
		},
		{
			name: "Sender error - attempts to send to all despite errors",
			event: models.ReleaseEvent{
				RepoID:   1,
				RepoName: "owner/repo",
				Tag:      "v1.0.0",
			},
			mockSubs: []models.Subscription{
				{Subscriber: &models.Subscriber{Email: "user1@example.com"}},
				{Subscriber: &models.Subscriber{Email: "user2@example.com"}},
			},
			repoErr:        nil,
			senderErr:      errors.New("smtp error"),
			expectedEmails: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockNotificationSubRepo{
				subs: tt.mockSubs,
				err:  tt.repoErr,
			}
			sender := &mockEmailSender{
				err: tt.senderErr,
			}
			builder := &mockMessageBuilder{}

			svc := NewNotificationService(log, repo, sender, builder)
			svc.processReleaseEvent(context.Background(), tt.event)

			if len(sender.sentEmails) != tt.expectedEmails {
				t.Errorf("expected %d emails attempted, got %d", tt.expectedEmails, len(sender.sentEmails))
			}
		})
	}
}

func TestNotificationService_StartAndStop(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	repo := &mockNotificationSubRepo{}
	sender := &mockEmailSender{}
	builder := &mockMessageBuilder{}

	svc := NewNotificationService(log, repo, sender, builder)

	eventsChan := make(chan models.ReleaseEvent)
	subsChan := make(chan models.SubscriptionEvent)
	ctx, cancel := context.WithCancel(context.Background())

	go svc.Start(ctx, eventsChan, subsChan)

	eventsChan <- models.ReleaseEvent{
		RepoID:   1,
		RepoName: "owner/test",
		Tag:      "v1",
	}

	time.Sleep(10 * time.Millisecond)

	cancel()

	time.Sleep(10 * time.Millisecond)
}
