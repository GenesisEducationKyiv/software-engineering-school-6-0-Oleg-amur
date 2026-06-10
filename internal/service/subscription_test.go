package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
)

func TestSubscribe(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tests := []struct {
		name          string
		req           model.SubscribeRequest
		sub           *model.Subscriber
		subErr        error
		repo          *model.Repository
		repoErr       error
		subCreateErr  error
		expectedError error
	}{
		{
			name:          "Subscriber service error",
			req:           model.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subErr:        errors.New("sub error"),
			expectedError: errors.New("sub error"),
		},
		{
			name:          "Repository service error",
			req:           model.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			sub:           &model.Subscriber{ID: 1},
			repoErr:       apperr.ErrRepoNotFound,
			expectedError: apperr.ErrRepoNotFound,
		},
		{
			name:          "Already subscribed",
			req:           model.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			sub:           &model.Subscriber{ID: 1},
			repo:          &model.Repository{ID: 1},
			subCreateErr:  apperr.ErrAlreadyExists,
			expectedError: apperr.ErrAlreadySubscribed,
		},
		{
			name:          "Success",
			req:           model.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			sub:           &model.Subscriber{ID: 1},
			repo:          &model.Repository{ID: 1},
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSubscriptionService(
				log,
				&mockSubscriberService{sub: tt.sub, err: tt.subErr},
				&mockRepositoryService{repo: tt.repo, err: tt.repoErr},
				&mockSubscriptionRepo{createErr: tt.subCreateErr},
				make(chan model.SubscriptionEvent, 10),
			)

			err := svc.Subscribe(context.Background(), tt.req)

			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}

func TestConfirm(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tests := []struct {
		name          string
		token         string
		expectedError error
		activateErr   error
	}{
		{
			name:          "Missing token",
			token:         "test token",
			expectedError: apperr.ErrTokenNotFound,
			activateErr:   apperr.ErrNotFound,
		},
		{
			name:          "Success",
			token:         "test token",
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSubscriptionService(
				log,
				&mockSubscriberService{},
				&mockRepositoryService{},
				&mockSubscriptionRepo{activateErr: tt.activateErr},
				make(chan model.SubscriptionEvent, 10),
			)

			err := svc.Confirm(context.Background(), tt.token)

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}
		})
	}
}

func TestUnsubscribe(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tests := []struct {
		name          string
		token         string
		expectedError error
		deleteErr     error
	}{
		{
			name:          "Error for DB",
			token:         "test token",
			expectedError: sql.ErrConnDone,
			deleteErr:     sql.ErrConnDone,
		},
		{
			name:          "Success",
			token:         "test token",
			expectedError: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSubscriptionService(
				log,
				&mockSubscriberService{},
				&mockRepositoryService{},
				&mockSubscriptionRepo{deleteErr: tt.deleteErr},
				make(chan model.SubscriptionEvent, 10),
			)

			err := svc.Unsubscribe(context.Background(), tt.token)

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}
		})
	}
}

func TestGetSubscriptions(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tests := []struct {
		name          string
		email         string
		mockSubs      []model.Subscription
		mockErr       error
		expectedError error
		expectedLen   int
	}{
		{
			name:          "DB Error",
			email:         "test@example.com",
			mockErr:       sql.ErrConnDone,
			expectedError: sql.ErrConnDone,
			expectedLen:   0,
		},
		{
			name:          "Success empty",
			email:         "test@example.com",
			mockSubs:      []model.Subscription{},
			expectedError: nil,
			expectedLen:   0,
		},
		{
			name:  "Success with data",
			email: "test@example.com",
			mockSubs: []model.Subscription{
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
			expectedError: nil,
			expectedLen:   2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSubscriptionService(
				log,
				&mockSubscriberService{},
				&mockRepositoryService{},
				&mockSubscriptionRepo{
					getActiveByEmailSubs: tt.mockSubs,
					getActiveByEmailErr:  tt.mockErr,
				},
				make(chan model.SubscriptionEvent, 10),
			)

			subs, err := svc.GetSubscriptions(context.Background(), tt.email)

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
			}
			if len(subs) != tt.expectedLen {
				t.Errorf("expected %d subscriptions, got %d", tt.expectedLen, len(subs))
			}
		})
	}
}

type mockSubscriberService struct {
	sub *model.Subscriber
	err error
}

func (m *mockSubscriberService) GetOrCreate(ctx context.Context, email string) (*model.Subscriber, error) {
	return m.sub, m.err
}

type mockRepositoryService struct {
	repo *model.Repository
	err  error
}

func (m *mockRepositoryService) GetOrCreate(ctx context.Context, repoName string) (*model.Repository, error) {
	return m.repo, m.err
}

type mockSubscriptionRepo struct {
	createErr            error
	activateErr          error
	deleteErr            error
	getActiveByEmailSubs []model.Subscription
	getActiveByEmailErr  error
}

func (m *mockSubscriptionRepo) Create(ctx context.Context, subID, repoID int, token string) error {
	return m.createErr
}

func (m *mockSubscriptionRepo) Activate(ctx context.Context, token string) error {
	return m.activateErr
}

func (m *mockSubscriptionRepo) DeleteByToken(ctx context.Context, token string) error {
	return m.deleteErr
}

func (m *mockSubscriptionRepo) GetActiveByEmail(
	ctx context.Context,
	email string,
) ([]model.Subscription, error) {
	return m.getActiveByEmailSubs, m.getActiveByEmailErr
}
