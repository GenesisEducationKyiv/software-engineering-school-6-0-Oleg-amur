package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	releasetrackerdomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

func assertErrorIs(t *testing.T, got error, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

func assertSubscriptionEvent(t *testing.T, publisher *mockNotificationPublisher, wantEmail string) {
	t.Helper()

	if len(publisher.confirmations) != 1 {
		t.Fatalf("got %d subscription events, want 1", len(publisher.confirmations))
	}
	event := publisher.confirmations[0]
	if event.Email != wantEmail {
		t.Errorf("got subscription event email %q, want %q", event.Email, wantEmail)
	}
	if event.Token == "" {
		t.Error("want subscription event token to be set")
	}
	if event.EventID == "" {
		t.Error("want subscription event id to be set")
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type mockSubscriberRegistration struct {
	subscriber *domain.Subscriber
	err        error
}

func (f *mockSubscriberRegistration) Execute(ctx context.Context, email string) (*domain.Subscriber, error) {
	return f.subscriber, f.err
}

type mockRepositoryTracker struct {
	repository *releasetrackerdomain.Repository
	err        error
}

func (f *mockRepositoryTracker) Execute(ctx context.Context, repoName string) (*releasetrackerdomain.Repository, error) {
	return f.repository, f.err
}

type mockNotificationPublisher struct {
	confirmations   []events.SubscriptionConfirmationRequested
	releases        []events.ReleaseNotificationRequested
	confirmationErr error
	releaseErr      error
}

func (f *mockNotificationPublisher) PublishSubscriptionConfirmation(
	ctx context.Context,
	event events.SubscriptionConfirmationRequested,
) error {
	f.confirmations = append(f.confirmations, event)
	return f.confirmationErr
}

func (f *mockNotificationPublisher) PublishReleaseNotification(
	ctx context.Context,
	event events.ReleaseNotificationRequested,
) error {
	f.releases = append(f.releases, event)
	return f.releaseErr
}

type mockSubscriptionRepo struct {
	createErr            error
	activateErr          error
	deleteErr            error
	getActiveByEmailSubs []domain.Subscription
	getActiveByEmailErr  error
}

func (f *mockSubscriptionRepo) Create(ctx context.Context, subID, repoID int, token string) error {
	return f.createErr
}

func (f *mockSubscriptionRepo) Activate(ctx context.Context, token string) error {
	return f.activateErr
}

func (f *mockSubscriptionRepo) DeleteByToken(ctx context.Context, token string) error {
	return f.deleteErr
}

func (f *mockSubscriptionRepo) GetActiveByEmail(
	ctx context.Context,
	email string,
) ([]domain.Subscription, error) {
	return f.getActiveByEmailSubs, f.getActiveByEmailErr
}
