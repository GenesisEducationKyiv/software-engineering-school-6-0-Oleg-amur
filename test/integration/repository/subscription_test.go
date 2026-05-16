//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/repository/postgresql"
)

type subscriptionFixture struct {
	subscriber *model.Subscriber
	repository *model.Repository
	token      string
}

func TestSubscriptionRepository_CreateRejectsDuplicateSubscriberRepository(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	subscriptionRepo, fixture := createSubscriptionFixture(t, ctx, "user@example.com", "owner/repo", "v1.0.0", "token-1")

	err := subscriptionRepo.Create(ctx, fixture.subscriber.ID, fixture.repository.ID, "token-2")

	if !errors.Is(err, apperr.ErrAlreadyExists) {
		t.Fatalf("got error %v, want %v", err, apperr.ErrAlreadyExists)
	}
}

func TestSubscriptionRepository_GetByTokenReturnsJoinedData(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	subscriptionRepo, fixture := createSubscriptionFixture(t, ctx, "user@example.com", "owner/repo", "v1.0.0", "token-1")

	got, err := subscriptionRepo.GetByToken(ctx, fixture.token)
	if err != nil {
		t.Fatalf("get subscription by token: %v", err)
	}

	assertSubscriptionJoin(t, got, fixture)
	if got.SubscriptionStatus != model.StatusPending {
		t.Fatalf("got subscription status %d, want %d", got.SubscriptionStatus, model.StatusPending)
	}
}

func TestSubscriptionRepository_GetByTokenReturnsNotFound(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	subscriptionRepo := postgresql.NewSubscriptionRepository(suite.DB)

	_, err := subscriptionRepo.GetByToken(ctx, "missing-token")

	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, apperr.ErrNotFound)
	}
}

func TestSubscriptionRepository_ActivateMakesSubscriptionVisibleByEmail(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	subscriptionRepo, fixture := createSubscriptionFixture(t, ctx, "user@example.com", "owner/repo", "v1.0.0", "token-1")

	beforeActivation, err := subscriptionRepo.GetActiveByEmail(ctx, fixture.subscriber.Email)
	if err != nil {
		t.Fatalf("get active subscriptions before activation: %v", err)
	}
	if len(beforeActivation) != 0 {
		t.Fatalf("got %d active subscriptions before activation, want 0", len(beforeActivation))
	}

	if err := subscriptionRepo.Activate(ctx, fixture.token); err != nil {
		t.Fatalf("activate subscription: %v", err)
	}

	active, err := subscriptionRepo.GetActiveByEmail(ctx, fixture.subscriber.Email)
	if err != nil {
		t.Fatalf("get active subscriptions by email: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("got %d active subscriptions by email, want 1", len(active))
	}
	if active[0].Repository.Name != fixture.repository.Name {
		t.Fatalf("got repository %q, want %q", active[0].Repository.Name, fixture.repository.Name)
	}
	if active[0].Repository.LastSeenTag != fixture.repository.LastSeenTag {
		t.Fatalf("got repository last seen tag %q, want %q", active[0].Repository.LastSeenTag, fixture.repository.LastSeenTag)
	}
}

func TestSubscriptionRepository_ActivateReturnsNotFound(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	subscriptionRepo := postgresql.NewSubscriptionRepository(suite.DB)

	err := subscriptionRepo.Activate(ctx, "missing-token")

	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, apperr.ErrNotFound)
	}
}

func TestSubscriptionRepository_GetActiveByRepoIDReturnsOnlyActiveSubscribers(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	subscriptionRepo, activeFixture := createSubscriptionFixture(t, ctx, "active@example.com", "owner/repo", "v1.0.0", "active-token")
	pendingFixture := createSubscriptionForRepository(t, ctx, activeFixture.repository, "pending@example.com", "pending-token")

	if err := subscriptionRepo.Activate(ctx, activeFixture.token); err != nil {
		t.Fatalf("activate subscription: %v", err)
	}

	active, err := subscriptionRepo.GetActiveByRepoID(ctx, activeFixture.repository.ID)
	if err != nil {
		t.Fatalf("get active subscriptions by repository id: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("got %d active subscriptions by repository id, want 1", len(active))
	}
	if active[0].Subscriber.Email != activeFixture.subscriber.Email {
		t.Fatalf("got subscriber email %q, want %q", active[0].Subscriber.Email, activeFixture.subscriber.Email)
	}
	if active[0].Subscriber.Email == pendingFixture.subscriber.Email {
		t.Fatalf("pending subscriber %q should not be returned", pendingFixture.subscriber.Email)
	}
}

func TestSubscriptionRepository_DeleteByTokenRemovesSubscription(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	subscriptionRepo, fixture := createSubscriptionFixture(t, ctx, "user@example.com", "owner/repo", "v1.0.0", "token-1")

	if err := subscriptionRepo.DeleteByToken(ctx, fixture.token); err != nil {
		t.Fatalf("delete subscription by token: %v", err)
	}
	_, err := subscriptionRepo.GetByToken(ctx, fixture.token)
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("got error %v after deleting subscription, want %v", err, apperr.ErrNotFound)
	}
}

func createSubscriptionFixture(
	t *testing.T,
	ctx context.Context,
	email string,
	repoName string,
	lastSeenTag string,
	token string,
) (*postgresql.SubscriptionRepository, subscriptionFixture) {
	t.Helper()

	subscriberRepo := postgresql.NewSubscriberRepository(suite.DB)
	repositoryRepo := postgresql.NewRepositoryRepository(suite.DB)
	subscriptionRepo := postgresql.NewSubscriptionRepository(suite.DB)

	subscriber, err := subscriberRepo.Create(ctx, email)
	if err != nil {
		t.Fatalf("create subscriber: %v", err)
	}
	repository, err := repositoryRepo.Create(ctx, repoName, lastSeenTag)
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	fixture := subscriptionFixture{
		subscriber: subscriber,
		repository: repository,
		token:      token,
	}

	if err := subscriptionRepo.Create(ctx, subscriber.ID, repository.ID, token); err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	return subscriptionRepo, fixture
}

func createSubscriptionForRepository(
	t *testing.T,
	ctx context.Context,
	repository *model.Repository,
	email string,
	token string,
) subscriptionFixture {
	t.Helper()

	subscriberRepo := postgresql.NewSubscriberRepository(suite.DB)
	subscriptionRepo := postgresql.NewSubscriptionRepository(suite.DB)

	subscriber, err := subscriberRepo.Create(ctx, email)
	if err != nil {
		t.Fatalf("create subscriber: %v", err)
	}
	if err := subscriptionRepo.Create(ctx, subscriber.ID, repository.ID, token); err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	return subscriptionFixture{
		subscriber: subscriber,
		repository: repository,
		token:      token,
	}
}

func assertSubscriptionJoin(t *testing.T, sub *model.Subscription, want subscriptionFixture) {
	t.Helper()

	if sub.SubscriberID != want.subscriber.ID {
		t.Fatalf("got subscriber id %d, want %d", sub.SubscriberID, want.subscriber.ID)
	}
	if sub.RepositoryID != want.repository.ID {
		t.Fatalf("got repository id %d, want %d", sub.RepositoryID, want.repository.ID)
	}
	if sub.Subscriber == nil {
		t.Fatal("want subscription subscriber to be populated")
	}
	if sub.Subscriber.Email != want.subscriber.Email {
		t.Fatalf("got subscriber email %q, want %q", sub.Subscriber.Email, want.subscriber.Email)
	}
	if sub.Repository == nil {
		t.Fatal("want subscription repository to be populated")
	}
	if sub.Repository.Name != want.repository.Name {
		t.Fatalf("got repository name %q, want %q", sub.Repository.Name, want.repository.Name)
	}
	if sub.Repository.LastSeenTag != want.repository.LastSeenTag {
		t.Fatalf("got repository last seen tag %q, want %q", sub.Repository.LastSeenTag, want.repository.LastSeenTag)
	}
}
