package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/domain"
	subscriptiondomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

func TestPlanReleaseNotifications_Execute(t *testing.T) {
	repo := &mockReleaseNotificationSubscriptionRepo{
		subscriptions: []subscriptiondomain.RepositorySubscription{
			{Email: "user1@example.com", UnsubscribeToken: "token-1"},
			{Email: "user2@example.com", UnsubscribeToken: "token-2"},
		},
	}
	publisher := &mockNotificationPublisher{}
	planner := NewPlanReleaseNotifications(testLogger(), repo, publisher)

	err := planner.Execute(context.Background(), domain.ReleaseEvent{
		RepoID:   10,
		RepoName: "owner/repo",
		Tag:      "v1.2.3",
	})
	if err != nil {
		t.Fatalf("got error %v, want nil", err)
	}
	if len(publisher.releases) != 2 {
		t.Fatalf("got %d release notifications, want 2", len(publisher.releases))
	}
	if publisher.releases[0].Email != "user1@example.com" {
		t.Errorf("got first email %q, want user1@example.com", publisher.releases[0].Email)
	}
	if publisher.releases[0].Repo != "owner/repo" {
		t.Errorf("got repo %q, want owner/repo", publisher.releases[0].Repo)
	}
	if publisher.releases[0].Tag != "v1.2.3" {
		t.Errorf("got tag %q, want v1.2.3", publisher.releases[0].Tag)
	}
	if publisher.releases[0].UnsubscribeToken != "token-1" {
		t.Errorf("got token %q, want token-1", publisher.releases[0].UnsubscribeToken)
	}
	if publisher.releases[0].EventID == "" {
		t.Error("want release notification event id to be set")
	}
	if publisher.releases[0].SchemaVersion != events.NotificationSchemaVersion {
		t.Errorf(
			"got schema version %d, want %d",
			publisher.releases[0].SchemaVersion,
			events.NotificationSchemaVersion,
		)
	}
}

func TestPlanReleaseNotifications_Execute_ReturnsRepositoryError(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &mockReleaseNotificationSubscriptionRepo{err: repoErr}
	publisher := &mockNotificationPublisher{}
	planner := NewPlanReleaseNotifications(testLogger(), repo, publisher)

	err := planner.Execute(context.Background(), domain.ReleaseEvent{RepoID: 10})

	if err == nil {
		t.Fatal("got nil error, want repository error")
	}
	if len(publisher.releases) != 0 {
		t.Errorf("got %d release notifications, want 0", len(publisher.releases))
	}
}

func TestPlanReleaseNotifications_Execute_ReturnsPublishError(t *testing.T) {
	repo := &mockReleaseNotificationSubscriptionRepo{
		subscriptions: []subscriptiondomain.RepositorySubscription{
			{Email: "user@example.com", UnsubscribeToken: "token"},
		},
	}
	publisher := &mockNotificationPublisher{releaseErr: errors.New("broker down")}
	planner := NewPlanReleaseNotifications(testLogger(), repo, publisher)

	err := planner.Execute(context.Background(), domain.ReleaseEvent{
		RepoID:   10,
		RepoName: "owner/repo",
		Tag:      "v1.2.3",
	})

	if err == nil {
		t.Fatal("got nil error, want publish error")
	}
	if len(publisher.releases) != 1 {
		t.Errorf("got %d release notifications, want 1", len(publisher.releases))
	}
}

type mockReleaseNotificationSubscriptionRepo struct {
	subscriptions []subscriptiondomain.RepositorySubscription
	err           error
}

func (f *mockReleaseNotificationSubscriptionRepo) Execute(
	ctx context.Context,
	repoID int,
) ([]subscriptiondomain.RepositorySubscription, error) {
	return f.subscriptions, f.err
}

type mockNotificationPublisher struct {
	releases   []events.ReleaseNotificationRequested
	releaseErr error
}

func (f *mockNotificationPublisher) PublishReleaseNotification(
	ctx context.Context,
	event events.ReleaseNotificationRequested,
) error {
	f.releases = append(f.releases, event)
	return f.releaseErr
}
