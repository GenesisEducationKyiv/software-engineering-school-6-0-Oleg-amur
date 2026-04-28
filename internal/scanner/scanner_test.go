package scanner

import (
	"context"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/models"
)

func TestScan(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tests := []struct {
		name                string
		mockRepos           []models.Repository
		mockSubs            []models.Subscription
		mockGithubTags      map[string]string
		expectedUpdateCount int
		expectedUpdatedTag  string
		expectedEmailCount  int
	}{
		{
			name: "No new release",
			mockRepos: []models.Repository{
				{ID: 1, Name: "owner/repo", LastSeenTag: "v1.0.0"},
			},
			mockSubs: nil,
			mockGithubTags: map[string]string{
				"owner/repo": "v1.0.0",
			},
			expectedUpdateCount: 0,
			expectedEmailCount:  0,
		},
		{
			name: "New release found",
			mockRepos: []models.Repository{
				{ID: 1, Name: "owner/repo", LastSeenTag: "v1.0.0"},
			},
			mockSubs: []models.Subscription{
				{Subscriber: &models.Subscriber{Email: "test1@example.com"}},
				{Subscriber: &models.Subscriber{Email: "test2@example.com"}},
			},
			mockGithubTags: map[string]string{
				"owner/repo": "v2.0.0",
			},
			expectedUpdateCount: 1,
			expectedUpdatedTag:  "v2.0.0",
			expectedEmailCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRepo := &mockRepositoryRepo{
				repos: tt.mockRepos,
			}
			subRepo := &mockSubscriptionRepo{
				subs: tt.mockSubs,
			}
			ghClient := &mockGithubClient{
				tags: tt.mockGithubTags,
			}
			notifier := &mockNotifier{}

			s := NewScanner(log, repoRepo, subRepo, ghClient, notifier, time.Hour)
			s.Scan(context.Background())

			if len(repoRepo.updateArgs) != tt.expectedUpdateCount {
				t.Errorf("expected %d database updates, got %d", tt.expectedUpdateCount, len(repoRepo.updateArgs))
			}
			if tt.expectedUpdateCount > 0 && repoRepo.updateArgs[0].tag != tt.expectedUpdatedTag {
				t.Errorf("expected database tag to be updated to %s, got %s", tt.expectedUpdatedTag, repoRepo.updateArgs[0].tag)
			}
			if notifier.sentCount != tt.expectedEmailCount {
				t.Errorf("expected %d email notifications sent, got %d", tt.expectedEmailCount, notifier.sentCount)
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

type mockSubscriptionRepo struct {
	subs   []models.Subscription
	getErr error
}

func (m *mockSubscriptionRepo) GetActiveByRepoID(ctx context.Context, repoID int) ([]models.Subscription, error) {
	return m.subs, m.getErr
}

type mockNotifier struct {
	sentCount int
}

func (m *mockNotifier) SendReleaseNotification(ctx context.Context, email, repo, tag string) error {
	m.sentCount++
	return nil
}

type mockGithubClient struct {
	tags map[string]string
	errs map[string]error
}

func (m *mockGithubClient) GetRepositoryLatestTag(ctx context.Context, repoAddr string, log *slog.Logger) (string, error) {
	if err, ok := m.errs[repoAddr]; ok && err != nil {
		return "", err
	}
	return m.tags[repoAddr], nil
}

