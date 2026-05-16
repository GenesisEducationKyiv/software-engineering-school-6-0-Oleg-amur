package service

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
)

func TestSubscriptionService_Subscribe(t *testing.T) {
	errSubscriber := errors.New("subscriber service error")

	tests := []struct {
		name          string
		req           model.SubscribeRequest
		subscriber    *model.Subscriber
		subscriberErr error
		repository    *model.Repository
		repositoryErr error
		createErr     error
		wantErr       error
	}{
		{
			name:          "returns subscriber service error",
			req:           model.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriberErr: errSubscriber,
			wantErr:       errSubscriber,
		},
		{
			name:          "returns repository service error",
			req:           model.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriber:    &model.Subscriber{ID: 1},
			repositoryErr: apperr.ErrRepoNotFound,
			wantErr:       apperr.ErrRepoNotFound,
		},
		{
			name:       "returns already subscribed when repository rejects duplicate",
			req:        model.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriber: &model.Subscriber{ID: 1},
			repository: &model.Repository{ID: 1},
			createErr:  apperr.ErrAlreadyExists,
			wantErr:    apperr.ErrAlreadySubscribed,
		},
		{
			name:       "enqueues confirmation event after successful subscription",
			req:        model.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriber: &model.Subscriber{ID: 1},
			repository: &model.Repository{ID: 1},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := make(chan model.SubscriptionEvent, 10)
			svc := NewSubscriptionService(
				testLogger(),
				&mockSubscriberService{subscriber: tt.subscriber, err: tt.subscriberErr},
				&mockRepositoryService{repository: tt.repository, err: tt.repositoryErr},
				&mockSubscriptionRepo{createErr: tt.createErr},
				events,
			)

			err := svc.Subscribe(context.Background(), tt.req)

			assertErrorIs(t, err, tt.wantErr)
			if tt.wantErr == nil {
				assertSubscriptionEvent(t, events, tt.req.Email)
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
				make(chan model.SubscriptionEvent, 10),
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
				make(chan model.SubscriptionEvent, 10),
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
		subs    []model.Subscription
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
			subs:  []model.Subscription{},
		},
		{
			name:  "maps active subscriptions to DTOs",
			email: "test@example.com",
			subs: []model.Subscription{
				{
					SubscriptionStatus: model.StatusActive,
					Repository: &model.Repository{
						Name:        "owner/repo1",
						LastSeenTag: "v1.0",
					},
				},
				{
					SubscriptionStatus: model.StatusActive,
					Repository: &model.Repository{
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
				make(chan model.SubscriptionEvent, 10),
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

func assertSubscriptionEvent(t *testing.T, events <-chan model.SubscriptionEvent, wantEmail string) {
	t.Helper()

	select {
	case event := <-events:
		if event.Email != wantEmail {
			t.Fatalf("got subscription event email %q, want %q", event.Email, wantEmail)
		}
		if event.Token == "" {
			t.Fatal("want subscription event token to be set")
		}
	default:
		t.Fatal("want subscription event to be queued")
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type mockSubscriberService struct {
	subscriber *model.Subscriber
	err        error
}

func (f *mockSubscriberService) GetOrCreate(ctx context.Context, email string) (*model.Subscriber, error) {
	return f.subscriber, f.err
}

type mockRepositoryService struct {
	repository *model.Repository
	err        error
}

func (f *mockRepositoryService) GetOrCreate(ctx context.Context, repoName string) (*model.Repository, error) {
	return f.repository, f.err
}

type mockSubscriptionRepo struct {
	createErr            error
	activateErr          error
	deleteErr            error
	getActiveByEmailSubs []model.Subscription
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
) ([]model.Subscription, error) {
	return f.getActiveByEmailSubs, f.getActiveByEmailErr
}
