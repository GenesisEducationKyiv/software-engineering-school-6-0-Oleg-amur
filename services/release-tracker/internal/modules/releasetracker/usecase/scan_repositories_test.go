package usecase

import (
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/domain"
)

func TestScanRepositoriesPublishesNotificationsAndUpdatesTag(t *testing.T) {
	store := &repositoryStoreFake{
		repositories: []domain.Repository{{ID: 7, Name: "owner/repo", LastSeenTag: "v1"}},
	}
	github := &githubClientFake{latestTag: "v2"}
	subscriptions := &subscriptionClientFake{subscriptions: []domain.ActiveSubscription{
		{Email: "user@example.com", UnsubscribeToken: "token"},
	}}
	publisher := &notificationPublisherFake{}
	usecase := NewScanRepositories(testLogger(), store, github, subscriptions, publisher)

	usecase.Execute(t.Context())

	if subscriptions.repositoryID != 7 {
		t.Fatalf("repository ID = %d, want 7", subscriptions.repositoryID)
	}
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
