package scanner

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/model"
)

func TestScanner_Scan(t *testing.T) {
	tests := []struct {
		name        string
		repos       []model.Repository
		githubTags  map[string]string
		wantUpdates int
		wantTag     string
		wantEvents  int
	}{
		{
			name: "does nothing when latest tag has not changed",
			repos: []model.Repository{
				{ID: 1, Name: "owner/repo", LastSeenTag: "v1.0.0"},
			},
			githubTags: map[string]string{
				"owner/repo": "v1.0.0",
			},
		},
		{
			name: "updates tag and emits release event when new release is found",
			repos: []model.Repository{
				{ID: 1, Name: "owner/repo", LastSeenTag: "v1.0.0"},
			},
			githubTags: map[string]string{
				"owner/repo": "v2.0.0",
			},
			wantUpdates: 1,
			wantTag:     "v2.0.0",
			wantEvents:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repoRepo := &mockRepositoryRepo{
				repos: tt.repos,
			}
			ghClient := &mockGithubClient{
				tags: tt.githubTags,
			}

			handler := &mockReleaseDetectedHandler{}

			scanner := NewScanner(testLogger(), repoRepo, ghClient, handler)
			scanner.Scan(context.Background())

			assertUpdatesLen(t, repoRepo, tt.wantUpdates)
			if tt.wantUpdates > 0 {
				assertUpdatedTag(t, repoRepo, tt.wantTag)
			}
			assertReleaseEvents(t, handler.events, tt.wantEvents, tt.wantTag)
		})
	}
}

func TestScanner_Scan_GetAllErrorStopsScan(t *testing.T) {
	repoRepo := &mockRepositoryRepo{
		getAllErr: errors.New("db down"),
	}
	ghClient := &mockGithubClient{
		tags: map[string]string{},
	}
	handler := &mockReleaseDetectedHandler{}

	scanner := NewScanner(testLogger(), repoRepo, ghClient, handler)
	scanner.Scan(context.Background())

	assertUpdatesLen(t, repoRepo, 0)
	if len(handler.events) != 0 {
		t.Errorf("got %d release events, want 0", len(handler.events))
	}
}

func TestScanner_Scan_RateLimitStopsRemainingRepos(t *testing.T) {
	repoRepo := &mockRepositoryRepo{
		repos: []model.Repository{
			{ID: 1, Name: "owner/rate-limited", LastSeenTag: "v1.0.0"},
			{ID: 2, Name: "owner/next", LastSeenTag: "v1.0.0"},
		},
	}
	ghClient := &mockGithubClient{
		tags: map[string]string{
			"owner/next": "v2.0.0",
		},
		errs: map[string]error{
			"owner/rate-limited": apperr.ErrRateLimitExceeded,
		},
	}
	handler := &mockReleaseDetectedHandler{}

	scanner := NewScanner(testLogger(), repoRepo, ghClient, handler)
	scanner.Scan(context.Background())

	assertUpdatesLen(t, repoRepo, 0)
	if len(handler.events) != 0 {
		t.Errorf("got %d release events, want 0", len(handler.events))
	}
}

func TestScanner_Scan_GithubErrorContinuesNextRepo(t *testing.T) {
	repoRepo := &mockRepositoryRepo{
		repos: []model.Repository{
			{ID: 1, Name: "owner/fails", LastSeenTag: "v1.0.0"},
			{ID: 2, Name: "owner/next", LastSeenTag: "v1.0.0"},
		},
	}
	ghClient := &mockGithubClient{
		tags: map[string]string{
			"owner/next": "v2.0.0",
		},
		errs: map[string]error{
			"owner/fails": errors.New("github down"),
		},
	}
	handler := &mockReleaseDetectedHandler{}

	scanner := NewScanner(testLogger(), repoRepo, ghClient, handler)
	scanner.Scan(context.Background())

	assertUpdatesLen(t, repoRepo, 1)
	if repoRepo.updateArgs[0].id != 2 {
		t.Errorf("got updated repo id %d, want 2", repoRepo.updateArgs[0].id)
	}
	if len(handler.events) != 1 {
		t.Errorf("got %d release events, want 1", len(handler.events))
	}
}

func TestScanner_Scan_UpdateErrorKeepsPublishedEvent(t *testing.T) {
	repoRepo := &mockRepositoryRepo{
		repos: []model.Repository{
			{ID: 1, Name: "owner/repo", LastSeenTag: "v1.0.0"},
		},
		updateErrs: []error{errors.New("db down")},
	}
	ghClient := &mockGithubClient{
		tags: map[string]string{
			"owner/repo": "v2.0.0",
		},
	}
	handler := &mockReleaseDetectedHandler{}

	scanner := NewScanner(testLogger(), repoRepo, ghClient, handler)
	scanner.Scan(context.Background())

	assertUpdatesLen(t, repoRepo, 1)
	if len(handler.events) != 1 {
		t.Errorf("got %d release events after update error, want 1", len(handler.events))
	}
}

func TestScanner_Scan_ReleaseHandlerErrorDoesNotUpdateTag(t *testing.T) {
	repoRepo := &mockRepositoryRepo{
		repos: []model.Repository{
			{ID: 1, Name: "owner/repo", LastSeenTag: "v1.0.0"},
		},
	}
	ghClient := &mockGithubClient{
		tags: map[string]string{
			"owner/repo": "v2.0.0",
		},
	}
	handler := &mockReleaseDetectedHandler{err: errors.New("broker down")}

	scanner := NewScanner(testLogger(), repoRepo, ghClient, handler)
	scanner.Scan(context.Background())

	assertUpdatesLen(t, repoRepo, 0)
	if len(handler.events) != 1 {
		t.Errorf("got %d release events, want 1", len(handler.events))
	}
}

func assertUpdatesLen(t *testing.T, repo *mockRepositoryRepo, want int) {
	t.Helper()

	if len(repo.updateArgs) != want {
		t.Fatalf("got %d updates, want %d", len(repo.updateArgs), want)
	}
}

func assertUpdatedTag(t *testing.T, repo *mockRepositoryRepo, want string) {
	t.Helper()

	if repo.updateArgs[0].tag != want {
		t.Errorf("got updated database tag %q, want %q", repo.updateArgs[0].tag, want)
	}
}

func assertReleaseEvents(t *testing.T, events []model.ReleaseEvent, wantCount int, wantTag string) {
	t.Helper()

	for _, event := range events {
		if wantTag != "" && event.Tag != wantTag {
			t.Errorf("got event tag %q, want %q", event.Tag, wantTag)
		}
	}

	if len(events) != wantCount {
		t.Errorf("got %d events, want %d", len(events), wantCount)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

type mockRepositoryRepo struct {
	repos      []model.Repository
	getAllErr  error
	updateErrs []error
	updateArgs []struct {
		id  int
		tag string
	}
}

func (f *mockRepositoryRepo) GetAll(ctx context.Context) ([]model.Repository, error) {
	return f.repos, f.getAllErr
}

func (f *mockRepositoryRepo) UpdateTag(ctx context.Context, id int, tag string) error {
	f.updateArgs = append(f.updateArgs, struct {
		id  int
		tag string
	}{id, tag})
	if len(f.updateErrs) > 0 {
		err := f.updateErrs[0]
		f.updateErrs = f.updateErrs[1:]
		return err
	}
	return nil
}

type mockGithubClient struct {
	tags map[string]string
	errs map[string]error
}

func (f *mockGithubClient) GetRepositoryLatestTag(
	ctx context.Context,
	repoAddr string,
) (string, error) {
	if err, ok := f.errs[repoAddr]; ok && err != nil {
		return "", err
	}
	return f.tags[repoAddr], nil
}

type mockReleaseDetectedHandler struct {
	events []model.ReleaseEvent
	err    error
}

func (f *mockReleaseDetectedHandler) HandleReleaseDetected(ctx context.Context, event model.ReleaseEvent) error {
	f.events = append(f.events, event)
	return f.err
}
