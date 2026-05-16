//go:build integration

package repository_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/repository/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/test/integration/testkit"
)

var suite *testkit.Suite

func TestMain(m *testing.M) {
	os.Exit(testkit.Run(m, func(s *testkit.Suite) {
		suite = s
	}))
}

func TestRepositoryRepository(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	repo := postgresql.NewRepositoryRepository(suite.DB)

	created, err := repo.Create(ctx, "owner/repo", "v1.0.0")
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("want created repository to have an id")
	}
	if created.Name != "owner/repo" {
		t.Fatalf("got repository name %q, want %q", created.Name, "owner/repo")
	}
	if created.LastSeenTag != "v1.0.0" {
		t.Fatalf("got last seen tag %q, want %q", created.LastSeenTag, "v1.0.0")
	}

	got, err := repo.GetByName(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("get repository by name: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("got repository id %d, want %d", got.ID, created.ID)
	}

	if err := repo.UpdateTag(ctx, created.ID, "v2.0.0"); err != nil {
		t.Fatalf("update repository tag: %v", err)
	}
	updated, err := repo.GetByName(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("get updated repository by name: %v", err)
	}
	if updated.LastSeenTag != "v2.0.0" {
		t.Fatalf("got updated tag %q, want %q", updated.LastSeenTag, "v2.0.0")
	}

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("get all repositories: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("got %d repositories, want 1", len(all))
	}

	_, err = repo.GetByName(ctx, "missing/repo")
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, apperr.ErrNotFound)
	}
}

func TestSubscriberRepository(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	repo := postgresql.NewSubscriberRepository(suite.DB)

	created, err := repo.Create(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("create subscriber: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("want created subscriber to have an id")
	}
	if created.Email != "user@example.com" {
		t.Fatalf("got email %q, want %q", created.Email, "user@example.com")
	}

	got, err := repo.GetByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("get subscriber by email: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("got subscriber id %d, want %d", got.ID, created.ID)
	}

	_, err = repo.GetByEmail(ctx, "missing@example.com")
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, apperr.ErrNotFound)
	}
}

func TestSubscriptionRepository(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()

	subscriberRepo := postgresql.NewSubscriberRepository(suite.DB)
	repositoryRepo := postgresql.NewRepositoryRepository(suite.DB)
	subscriptionRepo := postgresql.NewSubscriptionRepository(suite.DB)

	subscriber, err := subscriberRepo.Create(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("create subscriber: %v", err)
	}
	repository, err := repositoryRepo.Create(ctx, "owner/repo", "v1.0.0")
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	if err := subscriptionRepo.Create(ctx, subscriber.ID, repository.ID, "token-1"); err != nil {
		t.Fatalf("create subscription: %v", err)
	}

	if err := subscriptionRepo.Create(ctx, subscriber.ID, repository.ID, "token-2"); !errors.Is(err, apperr.ErrAlreadyExists) {
		t.Fatalf("got error %v, want %v", err, apperr.ErrAlreadyExists)
	}

	byToken, err := subscriptionRepo.GetByToken(ctx, "token-1")
	if err != nil {
		t.Fatalf("get subscription by token: %v", err)
	}
	assertSubscriptionJoin(t, byToken, subscriber.ID, repository.ID, "user@example.com", "owner/repo", "v1.0.0")
	if byToken.SubscriptionStatus != model.StatusPending {
		t.Fatalf("got subscription status %d, want %d", byToken.SubscriptionStatus, model.StatusPending)
	}

	activeByEmail, err := subscriptionRepo.GetActiveByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("get active subscriptions before activation: %v", err)
	}
	if len(activeByEmail) != 0 {
		t.Fatalf("got %d active subscriptions before activation, want 0", len(activeByEmail))
	}

	if err := subscriptionRepo.Activate(ctx, "token-1"); err != nil {
		t.Fatalf("activate subscription: %v", err)
	}

	activeByEmail, err = subscriptionRepo.GetActiveByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("get active subscriptions by email: %v", err)
	}
	if len(activeByEmail) != 1 {
		t.Fatalf("got %d active subscriptions by email, want 1", len(activeByEmail))
	}
	if activeByEmail[0].Repository.Name != "owner/repo" {
		t.Fatalf("got repository %q, want %q", activeByEmail[0].Repository.Name, "owner/repo")
	}

	activeByRepo, err := subscriptionRepo.GetActiveByRepoID(ctx, repository.ID)
	if err != nil {
		t.Fatalf("get active subscriptions by repository id: %v", err)
	}
	if len(activeByRepo) != 1 {
		t.Fatalf("got %d active subscriptions by repository id, want 1", len(activeByRepo))
	}
	if activeByRepo[0].Subscriber.Email != "user@example.com" {
		t.Fatalf("got subscriber email %q, want %q", activeByRepo[0].Subscriber.Email, "user@example.com")
	}

	if err := subscriptionRepo.Activate(ctx, "missing-token"); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, apperr.ErrNotFound)
	}

	if err := subscriptionRepo.DeleteByToken(ctx, "token-1"); err != nil {
		t.Fatalf("delete subscription by token: %v", err)
	}
	_, err = subscriptionRepo.GetByToken(ctx, "token-1")
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("got error %v after deleting subscription, want %v", err, apperr.ErrNotFound)
	}
}

func assertSubscriptionJoin(
	t *testing.T,
	sub *model.Subscription,
	subscriberID int,
	repositoryID int,
	email string,
	repoName string,
	lastSeenTag string,
) {
	t.Helper()

	if sub.SubscriberID != subscriberID {
		t.Fatalf("got subscriber id %d, want %d", sub.SubscriberID, subscriberID)
	}
	if sub.RepositoryID != repositoryID {
		t.Fatalf("got repository id %d, want %d", sub.RepositoryID, repositoryID)
	}
	if sub.Subscriber == nil {
		t.Fatal("want subscription subscriber to be populated")
	}
	if sub.Subscriber.Email != email {
		t.Fatalf("got subscriber email %q, want %q", sub.Subscriber.Email, email)
	}
	if sub.Repository == nil {
		t.Fatal("want subscription repository to be populated")
	}
	if sub.Repository.Name != repoName {
		t.Fatalf("got repository name %q, want %q", sub.Repository.Name, repoName)
	}
	if sub.Repository.LastSeenTag != lastSeenTag {
		t.Fatalf("got repository last seen tag %q, want %q", sub.Repository.LastSeenTag, lastSeenTag)
	}
}
