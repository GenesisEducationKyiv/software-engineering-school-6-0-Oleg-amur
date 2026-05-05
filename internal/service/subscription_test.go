package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/models"
)

func TestSubscribe(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tests := []struct {
		name           string
		req            models.SubscribeRequest
		ghTag          string
		ghErr          error
		ghExists       bool
		checkExistsErr error
		subRepoSub     *models.Subscriber
		getByEmailErr  error
		createSubErr   error
		repoRepoRepo   *models.Repository
		getByNameErr   error
		createRepoErr  error
		subCreateErr   error
		expectedError  error
	}{
		{
			name:          "Invalid format",
			req:           models.SubscribeRequest{Email: "test@example.com", Repo: "invalidformat"},
			expectedError: apperr.ErrInvalidFormat,
		},
		{
			name:           "GitHub rate limit",
			req:            models.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			getByNameErr:   sql.ErrNoRows,
			checkExistsErr: apperr.ErrRateLimitExceeded,
			expectedError:  apperr.ErrRateLimitExceeded,
		},
		{
			name:          "Repository not found",
			req:           models.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			getByNameErr:  sql.ErrNoRows,
			ghExists:      false,
			expectedError: apperr.ErrRepoNotFound,
		},
		{
			name:          "Repository has no releases but exists",
			req:           models.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			getByNameErr:  sql.ErrNoRows,
			ghErr:         apperr.ErrRepoNotFound,
			ghExists:      true,
			subRepoSub:    &models.Subscriber{ID: 1, Email: "test@example.com"},
			repoRepoRepo:  &models.Repository{ID: 1, Name: "owner/repo"},
			expectedError: nil,
		},
		{
			name:          "Success (New Repo)",
			req:           models.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			getByNameErr:  sql.ErrNoRows,
			ghExists:      true,
			ghTag:         "v1.0.0",
			ghErr:         nil,
			subRepoSub:    &models.Subscriber{ID: 1, Email: "test@example.com"},
			repoRepoRepo:  &models.Repository{ID: 1, Name: "owner/repo"},
			expectedError: nil,
		},
		{
			name:          "Success (Existing Repo in DB)",
			req:           models.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			getByNameErr:  nil,
			repoRepoRepo:  &models.Repository{ID: 1, Name: "owner/repo"},
			subRepoSub:    &models.Subscriber{ID: 1, Email: "test@example.com"},
			expectedError: nil,
		},
		{
			name:          "Already subscribed",
			req:           models.SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			getByNameErr:  nil,
			repoRepoRepo:  &models.Repository{ID: 1, Name: "owner/repo"},
			subRepoSub:    &models.Subscriber{ID: 1, Email: "test@example.com"},
			subCreateErr:  apperr.ErrAlreadyExists,
			expectedError: apperr.ErrAlreadySubscribed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewSubscriptionService(
				log,
				&mockSubscriberRepo{
					sub:           tt.subRepoSub,
					getByEmailErr: tt.getByEmailErr,
					createSubErr:  tt.createSubErr,
				},
				&mockRepositoryRepo{
					repo:          tt.repoRepoRepo,
					getByNameErr:  tt.getByNameErr,
					createRepoErr: tt.createRepoErr,
				},
				&mockSubscriptionRepo{createErr: tt.subCreateErr},
				&mockNotifier{},
				&mockGithubClient{
					tag:            tt.ghTag,
					err:            tt.ghErr,
					exists:         tt.ghExists,
					checkExistsErr: tt.checkExistsErr,
				},
			)

			err := svc.Subscribe(context.Background(), tt.req)

			if !errors.Is(err, tt.expectedError) {
				t.Errorf("expected error %v, got %v", tt.expectedError, err)
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
			activateErr:   sql.ErrNoRows,
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
				&mockSubscriberRepo{},
				&mockRepositoryRepo{},
				&mockSubscriptionRepo{activateErr: tt.activateErr},
				&mockNotifier{},
				&mockGithubClient{},
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
				&mockSubscriberRepo{},
				&mockRepositoryRepo{},
				&mockSubscriptionRepo{deleteErr: tt.deleteErr},
				&mockNotifier{},
				&mockGithubClient{},
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
		mockSubs      []models.Subscription
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
			mockSubs:      []models.Subscription{},
			expectedError: nil,
			expectedLen:   0,
		},
		{
			name:  "Success with data",
			email: "test@example.com",
			mockSubs: []models.Subscription{
				{
					SubscriptionStatus: models.StatusActive,
					Repository: &models.Repository{
						Name:        "owner/repo1",
						LastSeenTag: "v1.0",
					},
				},
				{
					SubscriptionStatus: models.StatusActive,
					Repository: &models.Repository{
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
				&mockSubscriberRepo{},
				&mockRepositoryRepo{},
				&mockSubscriptionRepo{
					getActiveByEmailSubs: tt.mockSubs,
					getActiveByEmailErr:  tt.mockErr,
				},
				&mockNotifier{},
				&mockGithubClient{},
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

type mockSubscriberRepo struct {
	sub           *models.Subscriber
	getByEmailErr error
	createSubErr  error
}

func (m *mockSubscriberRepo) GetByEmail(
	ctx context.Context,
	email string,
) (*models.Subscriber, error) {
	return m.sub, m.getByEmailErr
}

func (m *mockSubscriberRepo) Create(ctx context.Context, email string) (*models.Subscriber, error) {
	return m.sub, m.createSubErr
}

type mockRepositoryRepo struct {
	repo          *models.Repository
	getByNameErr  error
	createRepoErr error
}

func (m *mockRepositoryRepo) GetByName(
	ctx context.Context,
	name string,
) (*models.Repository, error) {
	return m.repo, m.getByNameErr
}

func (m *mockRepositoryRepo) Create(
	ctx context.Context,
	name string,
	lastSeenTag string,
) (*models.Repository, error) {
	return m.repo, m.createRepoErr
}

type mockSubscriptionRepo struct {
	createErr            error
	activateErr          error
	deleteErr            error
	getActiveByEmailSubs []models.Subscription
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
) ([]models.Subscription, error) {
	return m.getActiveByEmailSubs, m.getActiveByEmailErr
}

type mockNotifier struct{}

func (m *mockNotifier) SendConfirmation(ctx context.Context, email, token string) error {
	return nil
}

func (m *mockNotifier) SendReleaseNotification(ctx context.Context, email, repo, tag string) error {
	return nil
}

type mockGithubClient struct {
	tag            string
	err            error
	exists         bool
	checkExistsErr error
}

func (m *mockGithubClient) GetRepositoryLatestTag(
	ctx context.Context,
	repoAddr string,
) (string, error) {
	return m.tag, m.err
}

func (m *mockGithubClient) CheckIfRepoExists(
	ctx context.Context,
	repoAddr string,
) (bool, error) {
	return m.exists, m.checkExistsErr
}
