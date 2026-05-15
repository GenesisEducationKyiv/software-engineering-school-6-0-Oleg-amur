package scanner

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
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

			releasesChan := make(chan model.ReleaseEvent, 10)

			scanner := NewScanner(testLogger(), repoRepo, ghClient, releasesChan)
			scanner.Scan(context.Background())
			close(releasesChan)

			assertUpdatesLen(t, repoRepo, tt.wantUpdates)
			if tt.wantUpdates > 0 {
				assertUpdatedTag(t, repoRepo, tt.wantTag)
			}
			assertReleaseEvents(t, releasesChan, tt.wantEvents, tt.wantTag)
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
	releasesChan := make(chan model.ReleaseEvent, 10)

	scanner := NewScanner(testLogger(), repoRepo, ghClient, releasesChan)
	scanner.Scan(context.Background())

	assertUpdatesLen(t, repoRepo, 0)
	if len(releasesChan) != 0 {
		t.Fatalf("expected no release events, got %d", len(releasesChan))
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
	releasesChan := make(chan model.ReleaseEvent, 10)

	scanner := NewScanner(testLogger(), repoRepo, ghClient, releasesChan)
	scanner.Scan(context.Background())

	assertUpdatesLen(t, repoRepo, 0)
	if len(releasesChan) != 0 {
		t.Fatalf("expected no release events, got %d", len(releasesChan))
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
	releasesChan := make(chan model.ReleaseEvent, 10)

	scanner := NewScanner(testLogger(), repoRepo, ghClient, releasesChan)
	scanner.Scan(context.Background())

	assertUpdatesLen(t, repoRepo, 1)
	if repoRepo.updateArgs[0].id != 2 {
		t.Fatalf("expected repo id 2 to be updated, got %d", repoRepo.updateArgs[0].id)
	}
	if len(releasesChan) != 1 {
		t.Fatalf("expected 1 release event, got %d", len(releasesChan))
	}
}

func TestScanner_Scan_UpdateErrorDoesNotEmitEvent(t *testing.T) {
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
	releasesChan := make(chan model.ReleaseEvent, 10)

	scanner := NewScanner(testLogger(), repoRepo, ghClient, releasesChan)
	scanner.Scan(context.Background())

	assertUpdatesLen(t, repoRepo, 1)
	if len(releasesChan) != 0 {
		t.Fatalf("expected no release events after update error, got %d", len(releasesChan))
	}
}

func assertUpdatesLen(t *testing.T, repo *mockRepositoryRepo, want int) {
	t.Helper()

	if len(repo.updateArgs) != want {
		t.Fatalf("expected %d updates, got %d", want, len(repo.updateArgs))
	}
}

func assertUpdatedTag(t *testing.T, repo *mockRepositoryRepo, want string) {
	t.Helper()

	if repo.updateArgs[0].tag != want {
		t.Fatalf("expected database tag to be updated to %s, got %s", want, repo.updateArgs[0].tag)
	}
}

func assertReleaseEvents(t *testing.T, events <-chan model.ReleaseEvent, wantCount int, wantTag string) {
	t.Helper()

	var gotCount int
	for event := range events {
		gotCount++
		if wantTag != "" && event.Tag != wantTag {
			t.Fatalf("expected event tag to be %s, got %s", wantTag, event.Tag)
		}
	}

	if gotCount != wantCount {
		t.Fatalf("expected %d events, got %d", wantCount, gotCount)
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
