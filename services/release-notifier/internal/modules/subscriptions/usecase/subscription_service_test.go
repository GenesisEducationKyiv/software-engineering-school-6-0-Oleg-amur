package usecase

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	subscriptiondto "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/dto"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
)

func TestSubscriptionService_Subscribe(t *testing.T) {
	errSubscriber := errors.New("subscriber service error")

	tests := []struct {
		name          string
		req           subscriptiondto.SubscribeRequest
		subscriber    *domain.Subscriber
		subscriberErr error
		repository    *domain.RepositoryRef
		repositoryErr error
		createErr     error
		publishErr    error
		wantErr       error
	}{
		{
			name:          "returns subscriber service error",
			req:           subscriptiondto.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriberErr: errSubscriber,
			wantErr:       errSubscriber,
		},
		{
			name:          "returns repository service error",
			req:           subscriptiondto.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriber:    &domain.Subscriber{ID: 1},
			repositoryErr: apperr.ErrRepoNotFound,
			wantErr:       apperr.ErrRepoNotFound,
		},
		{
			name:       "returns already subscribed when repository rejects duplicate",
			req:        subscriptiondto.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriber: &domain.Subscriber{ID: 1},
			repository: &domain.RepositoryRef{ID: 1},
			createErr:  apperr.ErrAlreadyExists,
			wantErr:    apperr.ErrAlreadySubscribed,
		},
		{
			name:       "enqueues confirmation event after successful subscription",
			req:        subscriptiondto.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriber: &domain.Subscriber{ID: 1},
			repository: &domain.RepositoryRef{ID: 1},
		},
		{
			name:       "returns publish error after successful subscription",
			req:        subscriptiondto.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriber: &domain.Subscriber{ID: 1},
			repository: &domain.RepositoryRef{ID: 1},
			publishErr: errors.New("broker down"),
			wantErr:    errors.New("broker down"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher := &mockNotificationPublisher{confirmationErr: tt.publishErr}
			svc := NewSubscriptionService(
				testLogger(),
				&mockSubscriberService{subscriber: tt.subscriber, err: tt.subscriberErr},
				&mockRepositoryService{repository: tt.repository, err: tt.repositoryErr},
				&mockSubscriptionRepo{createErr: tt.createErr},
				publisher,
			)

			err := svc.Subscribe(context.Background(), tt.req)

			if tt.wantErr == nil {
				assertErrorIs(t, err, nil)
				assertSubscriptionEvent(t, publisher, tt.req.Email)
			} else if err == nil {
				t.Fatalf("got nil error, want %v", tt.wantErr)
			}
		})
	}
}

func TestSubscriptionService_Confirm(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		activateErr error
		wantErr     error
	}{
		{
			name:        "returns token not found when repository has no token",
			token:       "test-token",
			activateErr: apperr.ErrNotFound,
			wantErr:     apperr.ErrTokenNotFound,
		},
		{
			name:  "confirms subscription",
			token: "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSubscriptionService(
				testLogger(),
				&mockSubscriberService{},
				&mockRepositoryService{},
				&mockSubscriptionRepo{activateErr: tt.activateErr},
				&mockNotificationPublisher{},
			)

			err := svc.Confirm(context.Background(), tt.token)

			assertErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestSubscriptionService_Unsubscribe(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		deleteErr error
		wantErr   error
	}{
		{
			name:      "returns repository delete error",
			token:     "test-token",
			deleteErr: sql.ErrConnDone,
			wantErr:   sql.ErrConnDone,
		},
		{
			name:  "deletes subscription by token",
			token: "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSubscriptionService(
				testLogger(),
				&mockSubscriberService{},
				&mockRepositoryService{},
				&mockSubscriptionRepo{deleteErr: tt.deleteErr},
				&mockNotificationPublisher{},
			)

			err := svc.Unsubscribe(context.Background(), tt.token)

			assertErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestSubscriptionService_GetSubscriptions(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		subs    []domain.Subscription
		repoErr error
		wantErr error
		wantLen int
	}{
		{
			name:    "returns repository query error",
			email:   "test@example.com",
			repoErr: sql.ErrConnDone,
			wantErr: sql.ErrConnDone,
		},
		{
			name:  "returns empty list when user has no active subscriptions",
			email: "test@example.com",
			subs:  []domain.Subscription{},
		},
		{
			name:  "maps active subscriptions to DTOs",
			email: "test@example.com",
			subs: []domain.Subscription{
				{
					SubscriptionStatus: domain.StatusActive,
					Repository: &domain.RepositoryRef{
						Name:        "owner/repo1",
						LastSeenTag: "v1.0",
					},
				},
				{
					SubscriptionStatus: domain.StatusActive,
					Repository: &domain.RepositoryRef{
						Name:        "owner/repo2",
						LastSeenTag: "v2.0",
					},
				},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSubscriptionService(
				testLogger(),
				&mockSubscriberService{},
				&mockRepositoryService{},
				&mockSubscriptionRepo{
					getActiveByEmailSubs: tt.subs,
					getActiveByEmailErr:  tt.repoErr,
				},
				&mockNotificationPublisher{},
			)

			subs, err := svc.GetSubscriptions(context.Background(), tt.email)

			assertErrorIs(t, err, tt.wantErr)
			if len(subs) != tt.wantLen {
				t.Errorf("got %d subscriptions, want %d", len(subs), tt.wantLen)
			}
		})
	}
}

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

type mockSubscriberService struct {
	subscriber *domain.Subscriber
	err        error
}

func (f *mockSubscriberService) GetOrCreate(ctx context.Context, email string) (*domain.Subscriber, error) {
	return f.subscriber, f.err
}

type mockRepositoryService struct {
	repository *domain.RepositoryRef
	err        error
}

func (f *mockRepositoryService) EnsureTracked(ctx context.Context, repoName string) (*domain.RepositoryRef, error) {
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
