package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/models"
)

func TestRepositoryService_GetOrCreate(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tests := []struct {
		name          string
		repoName      string
		mockRepo      *models.Repository
		getErr        error
		ghExists      bool
		ghCheckErr    error
		ghTag         string
		ghTagErr      error
		createErr     error
		expectedError error
	}{
		{
			name:          "Invalid format",
			repoName:      "invalid",
			expectedError: apperr.ErrInvalidFormat,
		},
		{
			name:     "Exists in DB",
			repoName: "owner/repo",
			mockRepo: &models.Repository{ID: 1, Name: "owner/repo"},
			getErr:   nil,
		},
		{
			name:          "GitHub rate limit",
			repoName:      "owner/repo",
			getErr:        apperr.ErrNotFound,
			ghCheckErr:    apperr.ErrRateLimitExceeded,
			expectedError: apperr.ErrRateLimitExceeded,
		},
		{
			name:          "Not found on GitHub",
			repoName:      "owner/repo",
			getErr:        apperr.ErrNotFound,
			ghExists:      false,
			expectedError: apperr.ErrRepoNotFound,
		},
		{
			name:     "Success full sync",
			repoName: "owner/repo",
			getErr:   apperr.ErrNotFound,
			ghExists: true,
			ghTag:    "v1.0.0",
			mockRepo: &models.Repository{ID: 1, Name: "owner/repo", LastSeenTag: "v1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRepo := &mockRepositoryRepoForService{
				repo:      tt.mockRepo,
				getErr:    tt.getErr,
				createErr: tt.createErr,
			}
			ghClient := &mockGithubClientForService{
				exists:   tt.ghExists,
				checkErr: tt.ghCheckErr,
				tag:      tt.ghTag,
				tagErr:   tt.ghTagErr,
			}
			s := NewRepositoryService(log, repoRepo, ghClient)

			repo, err := s.GetOrCreate(context.Background(), tt.repoName)

			if tt.expectedError != nil {
				if err == nil || !errors.Is(err, tt.expectedError) {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if repo.Name != tt.repoName {
					t.Errorf("expected name %s, got %s", tt.repoName, repo.Name)
				}
			}
		})
	}
}

type mockRepositoryRepoForService struct {
	repo      *models.Repository
	getErr    error
	createErr error
}

func (m *mockRepositoryRepoForService) GetByName(ctx context.Context, name string) (*models.Repository, error) {
	return m.repo, m.getErr
}

func (m *mockRepositoryRepoForService) Create(
	ctx context.Context,
	name string,
	tag string,
) (*models.Repository, error) {
	return m.repo, m.createErr
}

type mockGithubClientForService struct {
	exists   bool
	checkErr error
	tag      string
	tagErr   error
}

func (m *mockGithubClientForService) CheckIfRepoExists(ctx context.Context, repo string) (bool, error) {
	return m.exists, m.checkErr
}

func (m *mockGithubClientForService) GetRepositoryLatestTag(ctx context.Context, repo string) (string, error) {
	return m.tag, m.tagErr
}
