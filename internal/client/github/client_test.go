package github

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
)

func TestClient_CheckIfRepoExists(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantExists bool
		wantErr    error
	}{
		{
			name:       "exists",
			statusCode: http.StatusOK,
			wantExists: true,
		},
		{
			name:       "not found",
			statusCode: http.StatusNotFound,
			wantExists: false,
		},
		{
			name:       "rate limited",
			statusCode: http.StatusTooManyRequests,
			wantErr:    apperr.ErrRateLimitExceeded,
		},
		{
			name:       "unexpected status",
			statusCode: http.StatusInternalServerError,
			wantErr:    errAny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				assertGithubHeaders(t, r)
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			client := NewClient(server.Client(), server.URL, "token", testLogger())
			gotExists, err := client.CheckIfRepoExists(context.Background(), "owner/repo")

			if gotPath != "/repos/owner/repo" {
				t.Errorf("got path %q, want %q", gotPath, "/repos/owner/repo")
			}
			if gotExists != tt.wantExists {
				t.Errorf("got exists %v, want %v", gotExists, tt.wantExists)
			}
			assertError(t, err, tt.wantErr)
		})
	}
}

func TestClient_GetRepositoryLatestTag(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantTag    string
		wantErr    error
	}{
		{
			name:       "returns latest tag",
			statusCode: http.StatusOK,
			body:       `{"tag_name":"v1.2.3"}`,
			wantTag:    "v1.2.3",
		},
		{
			name:       "repo not found",
			statusCode: http.StatusNotFound,
			wantErr:    apperr.ErrRepoNotFound,
		},
		{
			name:       "rate limited",
			statusCode: http.StatusTooManyRequests,
			wantErr:    apperr.ErrRateLimitExceeded,
		},
		{
			name:       "unexpected status",
			statusCode: http.StatusInternalServerError,
			wantErr:    errAny,
		},
		{
			name:       "invalid json",
			statusCode: http.StatusOK,
			body:       `{`,
			wantErr:    errAny,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotPath string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotPath = r.URL.Path
				assertGithubHeaders(t, r)
				w.WriteHeader(tt.statusCode)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer server.Close()

			client := NewClient(server.Client(), server.URL, "token", testLogger())
			gotTag, err := client.GetRepositoryLatestTag(context.Background(), "owner/repo")

			if gotPath != "/repos/owner/repo/releases/latest" {
				t.Errorf("got path %q, want %q", gotPath, "/repos/owner/repo/releases/latest")
			}
			if gotTag != tt.wantTag {
				t.Errorf("got tag %q, want %q", gotTag, tt.wantTag)
			}
			assertError(t, err, tt.wantErr)
		})
	}
}

func assertGithubHeaders(t *testing.T, r *http.Request) {
	t.Helper()

	if r.Header.Get(headerAccept) != acceptValue {
		t.Errorf("got Accept header %q, want %q", r.Header.Get(headerAccept), acceptValue)
	}
	if r.Header.Get(headerGitHubApiVersion) != apiVersionValue {
		t.Errorf(
			"got GitHub API version header %q, want %q",
			r.Header.Get(headerGitHubApiVersion),
			apiVersionValue,
		)
	}
	if r.Header.Get(headerAuthorization) != "Bearer token" {
		t.Errorf("got Authorization header %q, want %q", r.Header.Get(headerAuthorization), "Bearer token")
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

var errAny = errors.New("any error")

func assertError(t *testing.T, got error, want error) {
	t.Helper()

	if want == nil {
		if got != nil {
			t.Errorf("got error %v, want nil", got)
		}
		return
	}
	if errors.Is(want, errAny) {
		if got == nil {
			t.Error("got nil error, want non-nil error")
		}
		return
	}
	if !errors.Is(got, want) {
		t.Errorf("got error %v, want %v", got, want)
	}
}
