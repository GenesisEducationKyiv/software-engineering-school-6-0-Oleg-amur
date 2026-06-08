package service

import (
	"context"
	"errors"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
)

func TestReleaseNotificationPlanner_HandleReleaseDetected(t *testing.T) {
	repo := &mockReleaseNotificationSubscriptionRepo{
		subs: []model.Subscription{
			{Subscriber: &model.Subscriber{Email: "user1@example.com"}, Token: "token-1"},
			{Subscriber: &model.Subscriber{Email: "user2@example.com"}, Token: "token-2"},
		},
	}
	publisher := &mockNotificationPublisher{}
	planner := NewReleaseNotificationPlanner(testLogger(), repo, publisher)

	err := planner.HandleReleaseDetected(context.Background(), model.ReleaseEvent{
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
}

func TestReleaseNotificationPlanner_HandleReleaseDetected_ReturnsRepositoryError(t *testing.T) {
	repoErr := errors.New("db down")
	repo := &mockReleaseNotificationSubscriptionRepo{err: repoErr}
	publisher := &mockNotificationPublisher{}
	planner := NewReleaseNotificationPlanner(testLogger(), repo, publisher)

	err := planner.HandleReleaseDetected(context.Background(), model.ReleaseEvent{RepoID: 10})

	if err == nil {
		t.Fatal("got nil error, want repository error")
	}
	if len(publisher.releases) != 0 {
		t.Errorf("got %d release notifications, want 0", len(publisher.releases))
	}
}

func TestReleaseNotificationPlanner_HandleReleaseDetected_ReturnsPublishError(t *testing.T) {
	repo := &mockReleaseNotificationSubscriptionRepo{
		subs: []model.Subscription{
			{Subscriber: &model.Subscriber{Email: "user@example.com"}, Token: "token"},
		},
	}
	publisher := &mockNotificationPublisher{releaseErr: errors.New("broker down")}
	planner := NewReleaseNotificationPlanner(testLogger(), repo, publisher)

	err := planner.HandleReleaseDetected(context.Background(), model.ReleaseEvent{
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
	subs []model.Subscription
	err  error
}

func (f *mockReleaseNotificationSubscriptionRepo) GetActiveByRepoID(
	ctx context.Context,
	repoID int,
) ([]model.Subscription, error) {
	return f.subs, f.err
}
