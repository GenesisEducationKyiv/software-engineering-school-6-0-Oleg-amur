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

func TestRepositoryRepository_Integration(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	repo := postgresql.NewRepositoryRepository(suite.DB)

	created, err := repo.Create(ctx, "owner/repo", "v1.0.0")
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected created repository to have an id")
	}
	if created.Name != "owner/repo" {
		t.Fatalf("expected repository name owner/repo, got %s", created.Name)
	}
	if created.LastSeenTag != "v1.0.0" {
		t.Fatalf("expected last seen tag v1.0.0, got %s", created.LastSeenTag)
	}

	got, err := repo.GetByName(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("get repository by name: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected repository id %d, got %d", created.ID, got.ID)
	}

	if err := repo.UpdateTag(ctx, created.ID, "v2.0.0"); err != nil {
		t.Fatalf("update repository tag: %v", err)
	}
	updated, err := repo.GetByName(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("get updated repository by name: %v", err)
	}
	if updated.LastSeenTag != "v2.0.0" {
		t.Fatalf("expected updated tag v2.0.0, got %s", updated.LastSeenTag)
	}

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("get all repositories: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 repository, got %d", len(all))
	}

	_, err = repo.GetByName(ctx, "missing/repo")
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing repository, got %v", err)
	}
}

func TestSubscriberRepository_Integration(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	repo := postgresql.NewSubscriberRepository(suite.DB)

	created, err := repo.Create(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("create subscriber: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("expected created subscriber to have an id")
	}
	if created.Email != "user@example.com" {
		t.Fatalf("expected email user@example.com, got %s", created.Email)
	}

	got, err := repo.GetByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("get subscriber by email: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("expected subscriber id %d, got %d", created.ID, got.ID)
	}

	_, err = repo.GetByEmail(ctx, "missing@example.com")
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing subscriber, got %v", err)
	}
}

func TestSubscriptionRepository_Integration(t *testing.T) {
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
		t.Fatalf("expected ErrAlreadyExists for duplicate subscription, got %v", err)
	}

	byToken, err := subscriptionRepo.GetByToken(ctx, "token-1")
	if err != nil {
		t.Fatalf("get subscription by token: %v", err)
	}
	assertSubscriptionJoin(t, byToken, subscriber.ID, repository.ID, "user@example.com", "owner/repo", "v1.0.0")
	if byToken.SubscriptionStatus != model.StatusPending {
		t.Fatalf("expected pending subscription status, got %d", byToken.SubscriptionStatus)
	}

	activeByEmail, err := subscriptionRepo.GetActiveByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("get active subscriptions before activation: %v", err)
	}
	if len(activeByEmail) != 0 {
		t.Fatalf("expected no active subscriptions before activation, got %d", len(activeByEmail))
	}

	if err := subscriptionRepo.Activate(ctx, "token-1"); err != nil {
		t.Fatalf("activate subscription: %v", err)
	}

	activeByEmail, err = subscriptionRepo.GetActiveByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("get active subscriptions by email: %v", err)
	}
	if len(activeByEmail) != 1 {
		t.Fatalf("expected 1 active subscription by email, got %d", len(activeByEmail))
	}
	if activeByEmail[0].Repository.Name != "owner/repo" {
		t.Fatalf("expected repository owner/repo, got %s", activeByEmail[0].Repository.Name)
	}

	activeByRepo, err := subscriptionRepo.GetActiveByRepoID(ctx, repository.ID)
	if err != nil {
		t.Fatalf("get active subscriptions by repository id: %v", err)
	}
	if len(activeByRepo) != 1 {
		t.Fatalf("expected 1 active subscription by repository id, got %d", len(activeByRepo))
	}
	if activeByRepo[0].Subscriber.Email != "user@example.com" {
		t.Fatalf("expected subscriber email user@example.com, got %s", activeByRepo[0].Subscriber.Email)
	}

	if err := subscriptionRepo.Activate(ctx, "missing-token"); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for missing token activation, got %v", err)
	}

	if err := subscriptionRepo.DeleteByToken(ctx, "token-1"); err != nil {
		t.Fatalf("delete subscription by token: %v", err)
	}
	_, err = subscriptionRepo.GetByToken(ctx, "token-1")
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("expected ErrNotFound after deleting subscription, got %v", err)
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
		t.Fatalf("expected subscriber id %d, got %d", subscriberID, sub.SubscriberID)
	}
	if sub.RepositoryID != repositoryID {
		t.Fatalf("expected repository id %d, got %d", repositoryID, sub.RepositoryID)
	}
	if sub.Subscriber == nil {
		t.Fatal("expected subscription subscriber to be populated")
	}
	if sub.Subscriber.Email != email {
		t.Fatalf("expected subscriber email %s, got %s", email, sub.Subscriber.Email)
	}
	if sub.Repository == nil {
		t.Fatal("expected subscription repository to be populated")
	}
	if sub.Repository.Name != repoName {
		t.Fatalf("expected repository name %s, got %s", repoName, sub.Repository.Name)
	}
	if sub.Repository.LastSeenTag != lastSeenTag {
		t.Fatalf("expected repository last seen tag %s, got %s", lastSeenTag, sub.Repository.LastSeenTag)
	}
}
