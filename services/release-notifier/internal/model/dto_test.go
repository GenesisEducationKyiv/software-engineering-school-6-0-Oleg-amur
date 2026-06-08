package model

import (
	"errors"
	"strings"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
)

func TestSubscribeRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request SubscribeRequest
		wantErr error
	}{
		{
			name: "accepts valid request",
			request: SubscribeRequest{
				Email: "user@example.com",
				Repo:  "owner/repo",
			},
		},
		{
			name: "returns invalid format when email is invalid",
			request: SubscribeRequest{
				Email: "invalid-email",
				Repo:  "owner/repo",
			},
			wantErr: apperr.ErrInvalidFormat,
		},
		{
			name: "returns invalid format when repo is invalid",
			request: SubscribeRequest{
				Email: "user@example.com",
				Repo:  "owner",
			},
			wantErr: apperr.ErrInvalidFormat,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.request.Validate()

			assertErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{
			name:  "accepts valid email",
			email: "user@example.com",
			want:  true,
		},
		{
			name:  "rejects invalid email",
			email: "invalid-email",
		},
		{
			name:  "rejects display name",
			email: "User <user@example.com>",
		},
		{
			name:  "rejects too long email",
			email: strings.Repeat("a", maxEmailLength) + "@example.com",
		},
		{
			name:  "rejects too long local part",
			email: strings.Repeat("a", maxEmailLocalLength+1) + "@example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidEmail(tt.email)

			assertBool(t, got, tt.want)
		})
	}
}

func TestIsValidGitHubRepoPath(t *testing.T) {
	tests := []struct {
		name string
		repo string
		want bool
	}{
		{
			name: "accepts valid repo path",
			repo: "owner/repo",
			want: true,
		},
		{
			name: "rejects empty repo path",
			repo: "",
		},
		{
			name: "rejects repo path without slash",
			repo: "owner",
		},
		{
			name: "rejects repo path with extra parts",
			repo: "owner/repo/extra",
		},
		{
			name: "rejects empty owner",
			repo: "/repo",
		},
		{
			name: "rejects empty repo name",
			repo: "owner/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidGitHubRepoPath(tt.repo)

			assertBool(t, got, tt.want)
		})
	}
}

func TestIsValidGitHubOwner(t *testing.T) {
	tests := []struct {
		name  string
		owner string
		want  bool
	}{
		{
			name:  "accepts alphanumeric owner",
			owner: "owner123",
			want:  true,
		},
		{
			name:  "accepts owner with single hyphen",
			owner: "owner-name",
			want:  true,
		},
		{
			name:  "rejects owner with underscore",
			owner: "owner_name",
		},
		{
			name:  "rejects owner with space",
			owner: "owner ",
		},
		{
			name:  "rejects owner starting with hyphen",
			owner: "-owner",
		},
		{
			name:  "rejects owner ending with hyphen",
			owner: "owner-",
		},
		{
			name:  "rejects owner with consecutive hyphens",
			owner: "own--er",
		},
		{
			name:  "rejects too long owner",
			owner: strings.Repeat("a", maxGitHubOwnerLength+1),
		},
		{
			name:  "rejects non-ASCII owner",
			owner: "тестування",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidGitHubOwner(tt.owner)

			assertBool(t, got, tt.want)
		})
	}
}

func TestIsValidGitHubRepoName(t *testing.T) {
	tests := []struct {
		name     string
		repoName string
		want     bool
	}{
		{
			name:     "accepts alphanumeric repo name",
			repoName: "repo123",
			want:     true,
		},
		{
			name:     "accepts supported symbols",
			repoName: "repo.name_1-2",
			want:     true,
		},
		{
			name:     "accepts dot-prefixed repo name",
			repoName: ".github",
			want:     true,
		},
		{
			name:     "rejects empty repo name",
			repoName: "",
		},
		{
			name:     "rejects repo name with space",
			repoName: "repo name",
		},
		{
			name:     "rejects unsupported symbol",
			repoName: "repo+name",
		},
		{
			name:     "rejects control characters",
			repoName: "repo\n\t",
		},
		{
			name:     "rejects reserved dot",
			repoName: ".",
		},
		{
			name:     "rejects reserved dot dot",
			repoName: "..",
		},
		{
			name:     "rejects git suffix",
			repoName: "repo.git",
		},
		{
			name:     "rejects too long repo name",
			repoName: strings.Repeat("a", maxGitHubRepoLength+1),
		},
		{
			name:     "rejects non-ASCII repo name",
			repoName: "тестування",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidGitHubRepoName(tt.repoName)

			assertBool(t, got, tt.want)
		})
	}
}

func assertErrorIs(t *testing.T, got error, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Errorf("got error %v, want %v", got, want)
	}
}

func assertBool(t *testing.T, got bool, want bool) {
	t.Helper()

	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}
