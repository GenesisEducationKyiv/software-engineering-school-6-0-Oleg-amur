package usecase

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
)

func assertErrorIs(t *testing.T, got error, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}

func assertSubscriptionSagaStarted(t *testing.T, repo *mockSubscriptionRepo, wantEmail string) {
	t.Helper()

	if len(repo.startedConfirmations) != 1 {
		t.Fatalf("got %d started subscription sagas, want 1", len(repo.startedConfirmations))
	}
	started := repo.startedConfirmations[0]
	if started.email != wantEmail {
		t.Errorf("got saga email %q, want %q", started.email, wantEmail)
	}
	if started.token == "" {
		t.Error("want saga confirmation token to be set")
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
	repository *RepositoryView
	err        error
}

func (f *mockRepositoryTracker) EnsureTracked(
	ctx context.Context,
	repoName string,
) (*RepositoryView, error) {
	return f.repository, f.err
}

func (f *mockRepositoryTracker) GetRepository(
	ctx context.Context,
	repoName string,
) (*RepositoryView, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.repository != nil {
		return f.repository, nil
	}
	return &RepositoryView{Name: repoName}, nil
}

type mockSubscriptionRepo struct {
	createErr            error
	activateErr          error
	deleteErr            error
	getActiveByEmailSubs []domain.Subscription
	getActiveByEmailErr  error
	startedConfirmations []startedConfirmation
}

type startedConfirmation struct {
	subID    int
	repoName string
	email    string
	token    string
}

func (f *mockSubscriptionRepo) StartSubscriptionConfirmation(
	ctx context.Context,
	subID int,
	repoName, email string,
	token string,
) error {
	f.startedConfirmations = append(f.startedConfirmations, startedConfirmation{
		subID:    subID,
		repoName: repoName,
		email:    email,
		token:    token,
	})
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
