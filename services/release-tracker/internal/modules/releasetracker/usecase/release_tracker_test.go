package usecase

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/domain"
	repositorypostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/persistence/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

func TestEnsureTrackedCreatesMissingRepository(t *testing.T) {
	store := &repositoryStoreFake{getErr: repositorypostgresql.ErrNotFound}
	github := &githubFake{exists: true, latestTag: "v1.0.0"}
	tracker := New(testLogger(), store, github, &subscriptionsFake{}, &publisherFake{})

	tracked, err := tracker.EnsureTracked(t.Context(), "owner/repo")
	if err != nil {
		t.Fatalf("ensure repository: %v", err)
	}
	if tracked.Name != "owner/repo" || tracked.LastSeenTag != "v1.0.0" {
		t.Fatalf("unexpected repository: %+v", tracked)
	}
	if store.created != "owner/repo" {
		t.Fatalf("created repository %q", store.created)
	}
}

func TestScanPublishesNotificationsAndUpdatesTag(t *testing.T) {
	store := &repositoryStoreFake{all: []domain.Repository{{ID: 7, Name: "owner/repo", LastSeenTag: "v1"}}}
	github := &githubFake{latestTag: "v2"}
	subscriptions := &subscriptionsFake{subscriptions: []domain.ActiveSubscription{
		{Email: "user@example.com", UnsubscribeToken: "token"},
	}}
	publisher := &publisherFake{}
	tracker := New(testLogger(), store, github, subscriptions, publisher)

	tracker.Scan(t.Context())

	if len(publisher.events) != 1 {
		t.Fatalf("published %d events, want 1", len(publisher.events))
	}
	if publisher.events[0].Email != "user@example.com" || publisher.events[0].Tag != "v2" {
		t.Fatalf("unexpected event: %+v", publisher.events[0])
	}
	if store.updatedID != 7 || store.updatedTag != "v2" {
		t.Fatalf("unexpected tag update: id=%d tag=%q", store.updatedID, store.updatedTag)
	}
}

type repositoryStoreFake struct {
	get        *domain.Repository
	getErr     error
	all        []domain.Repository
	created    string
	updatedID  int
	updatedTag string
}

func (f *repositoryStoreFake) Create(_ context.Context, name, tag string) (*domain.Repository, error) {
	f.created = name
	return &domain.Repository{Name: name, LastSeenTag: tag}, nil
}

func (f *repositoryStoreFake) GetByName(context.Context, string) (*domain.Repository, error) {
	return f.get, f.getErr
}

func (f *repositoryStoreFake) GetAll(context.Context) ([]domain.Repository, error) { return f.all, nil }

func (f *repositoryStoreFake) UpdateTag(_ context.Context, id int, tag string) error {
	f.updatedID, f.updatedTag = id, tag
	return nil
}

type githubFake struct {
	exists    bool
	latestTag string
	err       error
}

func (f *githubFake) RepositoryExists(context.Context, string) (bool, error) {
	return f.exists, f.err
}

func (f *githubFake) LatestTag(context.Context, string) (string, error) {
	return f.latestTag, f.err
}

type subscriptionsFake struct {
	subscriptions []domain.ActiveSubscription
	err           error
}

func (f *subscriptionsFake) ListActiveByRepository(
	context.Context,
	string,
) ([]domain.ActiveSubscription, error) {
	return f.subscriptions, f.err
}

type publisherFake struct {
	events []events.ReleaseNotificationRequested
	err    error
}

func (f *publisherFake) Publish(_ context.Context, event events.ReleaseNotificationRequested) error {
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, event)
	return nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
