package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/domain"
)

func TestService_EnsureTracked(t *testing.T) {
	errRepository := errors.New("repository query error")
	errCreate := errors.New("repository create error")
	errGithub := errors.New("github error")

	tests := []struct {
		name       string
		repoName   string
		repository *domain.Repository
		getErr     error
		ghExists   bool
		ghCheckErr error
		ghTag      string
		ghTagErr   error
		createErr  error
		wantErr    error
	}{
		{
			name:       "returns repository when it already exists",
			repoName:   "owner/repo",
			repository: &domain.Repository{ID: 1, Name: "owner/repo"},
		},
		{
			name:     "returns wrapped repository error",
			repoName: "owner/repo",
			getErr:   errRepository,
			wantErr:  errRepository,
		},
		{
			name:       "returns rate limit when GitHub existence check is rate limited",
			repoName:   "owner/repo",
			getErr:     apperr.ErrNotFound,
			ghCheckErr: apperr.ErrRateLimitExceeded,
			wantErr:    apperr.ErrRateLimitExceeded,
		},
		{
			name:     "returns repo not found when GitHub repository does not exist",
			repoName: "owner/repo",
			getErr:   apperr.ErrNotFound,
			ghExists: false,
			wantErr:  apperr.ErrRepoNotFound,
		},
		{
			name:       "returns wrapped GitHub check error",
			repoName:   "owner/repo",
			getErr:     apperr.ErrNotFound,
			ghCheckErr: errGithub,
			wantErr:    errGithub,
		},
		{
			name:     "returns rate limit when GitHub latest tag is rate limited",
			repoName: "owner/repo",
			getErr:   apperr.ErrNotFound,
			ghExists: true,
			ghTagErr: apperr.ErrRateLimitExceeded,
			wantErr:  apperr.ErrRateLimitExceeded,
		},
		{
			name:       "creates repository when latest tag is unavailable because no release exists",
			repoName:   "owner/repo",
			getErr:     apperr.ErrNotFound,
			ghExists:   true,
			ghTagErr:   apperr.ErrRepoNotFound,
			repository: &domain.Repository{ID: 1, Name: "owner/repo"},
		},
		{
			name:     "returns wrapped GitHub latest tag error",
			repoName: "owner/repo",
			getErr:   apperr.ErrNotFound,
			ghExists: true,
			ghTagErr: errGithub,
			wantErr:  errGithub,
		},
		{
			name:      "returns wrapped create error",
			repoName:  "owner/repo",
			getErr:    apperr.ErrNotFound,
			ghExists:  true,
			ghTag:     "v1.0.0",
			createErr: errCreate,
			wantErr:   errCreate,
		},
		{
			name:       "creates repository with latest GitHub tag",
			repoName:   "owner/repo",
			getErr:     apperr.ErrNotFound,
			ghExists:   true,
			ghTag:      "v1.0.0",
			repository: &domain.Repository{ID: 1, Name: "owner/repo", LastSeenTag: "v1.0.0"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repositoryRepo := &mockRepositoryStore{
				repository: tt.repository,
				getErr:     tt.getErr,
				createErr:  tt.createErr,
			}
			ghClient := &mockRepositoryMetadataClient{
				exists:   tt.ghExists,
				checkErr: tt.ghCheckErr,
				tag:      tt.ghTag,
				tagErr:   tt.ghTagErr,
			}
			svc := NewRepositoryService(testLogger(), repositoryRepo, ghClient)

			repository, err := svc.EnsureTracked(context.Background(), tt.repoName)

			assertErrorIs(t, err, tt.wantErr)
			if tt.wantErr != nil {
				return
			}
			if repository.Name != tt.repoName {
				t.Errorf("got repository name %q, want %q", repository.Name, tt.repoName)
			}
		})
	}
}

type mockRepositoryStore struct {
	repository *domain.Repository
	getErr     error
	createErr  error
}

func (f *mockRepositoryStore) GetByName(ctx context.Context, name string) (*domain.Repository, error) {
	return f.repository, f.getErr
}

func (f *mockRepositoryStore) Create(
	ctx context.Context,
	name string,
	tag string,
) (*domain.Repository, error) {
	return f.repository, f.createErr
}

type mockRepositoryMetadataClient struct {
	exists   bool
	checkErr error
	tag      string
	tagErr   error
}

func (f *mockRepositoryMetadataClient) CheckIfRepoExists(ctx context.Context, repo string) (bool, error) {
	return f.exists, f.checkErr
}

func (f *mockRepositoryMetadataClient) GetRepositoryLatestTag(ctx context.Context, repo string) (string, error) {
	return f.tag, f.tagErr
}

func assertErrorIs(t *testing.T, got error, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}
