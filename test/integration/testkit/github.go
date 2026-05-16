//go:build integration

package testkit

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

type GitHubFixture struct {
	server *httptest.Server

	mu    sync.Mutex
	repos map[string]githubRepo
	calls map[string]int
}

type githubRepo struct {
	exists       bool
	tag          string
	repoStatus   int
	latestStatus int
	latestBody   string
}

func NewGitHubFixture(t testing.TB) *GitHubFixture {
	t.Helper()

	fixture := &GitHubFixture{
		repos: map[string]githubRepo{},
		calls: map[string]int{},
	}
	fixture.SetRepoExists("owner/repo", true)
	fixture.SetLatestTag("owner/repo", "v1.0.0")

	fixture.server = httptest.NewServer(http.HandlerFunc(fixture.handle))
	t.Cleanup(fixture.server.Close)

	return fixture
}

func (f *GitHubFixture) URL() string {
	return f.server.URL
}

func (f *GitHubFixture) Client() *http.Client {
	return f.server.Client()
}

func (f *GitHubFixture) SetRepoExists(repo string, exists bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	current := f.repos[repo]
	current.exists = exists
	f.repos[repo] = current
}

func (f *GitHubFixture) SetLatestTag(repo string, tag string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	current := f.repos[repo]
	current.tag = tag
	f.repos[repo] = current
}

func (f *GitHubFixture) SetRepoStatus(repo string, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	current := f.repos[repo]
	current.repoStatus = status
	f.repos[repo] = current
}

func (f *GitHubFixture) SetLatestReleaseStatus(repo string, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	current := f.repos[repo]
	current.latestStatus = status
	f.repos[repo] = current
}

func (f *GitHubFixture) SetLatestReleaseBody(repo string, body string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	current := f.repos[repo]
	current.latestBody = body
	f.repos[repo] = current
}

func (f *GitHubFixture) RequestCount(repo string) int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls[repo]
}

func (f *GitHubFixture) handle(w http.ResponseWriter, r *http.Request) {
	repo, latest, ok := parseGitHubRepoPath(r.URL.Path)
	if !ok || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	f.mu.Lock()
	f.calls[repo]++
	current, configured := f.repos[repo]
	f.mu.Unlock()

	if !configured {
		http.NotFound(w, r)
		return
	}

	if latest {
		f.handleLatestRelease(w, current)
		return
	}

	f.handleRepository(w, repo, current)
}

func (f *GitHubFixture) handleRepository(w http.ResponseWriter, repoName string, repo githubRepo) {
	if repo.repoStatus != 0 {
		w.WriteHeader(repo.repoStatus)
		return
	}
	if !repo.exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]string{"full_name": repoName})
}

func (f *GitHubFixture) handleLatestRelease(w http.ResponseWriter, repo githubRepo) {
	if repo.latestStatus != 0 {
		w.WriteHeader(repo.latestStatus)
		_, _ = w.Write([]byte(repo.latestBody))
		return
	}
	if !repo.exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	writeJSON(w, map[string]string{"tag_name": repo.tag})
}

func parseGitHubRepoPath(path string) (repo string, latest bool, ok bool) {
	path = strings.TrimPrefix(path, "/")
	if !strings.HasPrefix(path, "repos/") {
		return "", false, false
	}

	remainder := strings.TrimPrefix(path, "repos/")
	if strings.HasSuffix(remainder, "/releases/latest") {
		repo = strings.TrimSuffix(remainder, "/releases/latest")
		return repo, true, repo != ""
	}

	return remainder, false, remainder != ""
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(value)
}
