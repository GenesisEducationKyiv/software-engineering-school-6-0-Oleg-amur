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

type FakeGitHubServer struct {
	server *httptest.Server

	mu    sync.Mutex
	repos map[string]githubRepo
}

type githubRepo struct {
	exists     bool
	tag        string
	repoStatus int
}

func NewFakeGitHubServer(t testing.TB) *FakeGitHubServer {
	t.Helper()

	fake := &FakeGitHubServer{
		repos: map[string]githubRepo{
			"owner/repo": {exists: true, tag: "v1.0.0"},
		},
	}

	fake.server = httptest.NewServer(http.HandlerFunc(fake.handle))
	t.Cleanup(fake.server.Close)

	return fake
}

func (f *FakeGitHubServer) URL() string {
	return f.server.URL
}

func (f *FakeGitHubServer) SetRepoExists(repo string, exists bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	current := f.repos[repo]
	current.exists = exists
	f.repos[repo] = current
}

func (f *FakeGitHubServer) SetLatestTag(repo string, tag string) {
	f.mu.Lock()
	defer f.mu.Unlock()

	current := f.repos[repo]
	current.tag = tag
	f.repos[repo] = current
}

func (f *FakeGitHubServer) SetRepoStatus(repo string, status int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	current := f.repos[repo]
	current.repoStatus = status
	f.repos[repo] = current
}

func (f *FakeGitHubServer) handle(w http.ResponseWriter, r *http.Request) {
	repo, latest, ok := parseGitHubRepoPath(r.URL.Path)
	if !ok || r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}

	f.mu.Lock()
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

func (f *FakeGitHubServer) handleRepository(w http.ResponseWriter, repoName string, repo githubRepo) {
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

func (f *FakeGitHubServer) handleLatestRelease(w http.ResponseWriter, repo githubRepo) {
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
