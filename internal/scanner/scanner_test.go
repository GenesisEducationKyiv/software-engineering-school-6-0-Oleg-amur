package scanner

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/models"
)

func TestScan(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tests := []struct {
		name                string
		mockRepos           []models.Repository
		mockGithubTags      map[string]string
		expectedUpdateCount int
		expectedUpdatedTag  string
		expectedEventCount  int
	}{
		{
			name: "No new release",
			mockRepos: []models.Repository{
				{ID: 1, Name: "owner/repo", LastSeenTag: "v1.0.0"},
			},
			mockGithubTags: map[string]string{
				"owner/repo": "v1.0.0",
			},
			expectedUpdateCount: 0,
			expectedEventCount:  0,
		},
		{
			name: "New release found",
			mockRepos: []models.Repository{
				{ID: 1, Name: "owner/repo", LastSeenTag: "v1.0.0"},
			},
			mockGithubTags: map[string]string{
				"owner/repo": "v2.0.0",
			},
			expectedUpdateCount: 1,
			expectedUpdatedTag:  "v2.0.0",
			expectedEventCount:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRepo := &mockRepositoryRepo{
				repos: tt.mockRepos,
			}
			ghClient := &mockGithubClient{
				tags: tt.mockGithubTags,
			}
			
			releasesChan := make(chan models.ReleaseEvent, 10)

			s := NewScanner(log, repoRepo, ghClient, releasesChan)
			s.Scan(context.Background())
			close(releasesChan)

			if len(repoRepo.updateArgs) != tt.expectedUpdateCount {
				t.Errorf(
					"expected %d database updates, got %d",
					tt.expectedUpdateCount,
					len(repoRepo.updateArgs),
				)
			}
			if tt.expectedUpdateCount > 0 && repoRepo.updateArgs[0].tag != tt.expectedUpdatedTag {
				t.Errorf(
					"expected database tag to be updated to %s, got %s",
					tt.expectedUpdatedTag,
					repoRepo.updateArgs[0].tag,
				)
			}
			
			var actualEventCount int
			for event := range releasesChan {
				actualEventCount++
				if tt.expectedUpdateCount > 0 && event.Tag != tt.expectedUpdatedTag {
					t.Errorf(
						"expected event tag to be %s, got %s",
						tt.expectedUpdatedTag,
						event.Tag,
					)
				}
			}
			
			if actualEventCount != tt.expectedEventCount {
				t.Errorf(
					"expected %d events, got %d",
					tt.expectedEventCount,
					actualEventCount,
				)
			}
		})
	}
}

type mockRepositoryRepo struct {
	repos      []models.Repository
	getAllErr  error
	updateErrs []error
	updateArgs []struct {
		id  int
		tag string
	}
}

func (m *mockRepositoryRepo) GetAll(ctx context.Context) ([]models.Repository, error) {
	return m.repos, m.getAllErr
}

func (m *mockRepositoryRepo) UpdateTag(ctx context.Context, id int, tag string) error {
	m.updateArgs = append(m.updateArgs, struct {
		id  int
		tag string
	}{id, tag})
	if len(m.updateErrs) > 0 {
		err := m.updateErrs[0]
		m.updateErrs = m.updateErrs[1:]
		return err
	}
	return nil
}

type mockGithubClient struct {
	tags map[string]string
	errs map[string]error
}

func (m *mockGithubClient) GetRepositoryLatestTag(
	ctx context.Context,
	repoAddr string,
) (string, error) {
	if err, ok := m.errs[repoAddr]; ok && err != nil {
		return "", err
	}
	return m.tags[repoAddr], nil
}
