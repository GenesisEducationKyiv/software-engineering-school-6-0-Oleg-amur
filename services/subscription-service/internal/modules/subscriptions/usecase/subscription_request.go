package usecase

import (
	"net/mail"
	"regexp"
	"strings"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/apperr"
)

const (
	maxEmailLength       = 254
	maxEmailLocalLength  = 64
	maxGitHubOwnerLength = 39
	maxGitHubRepoLength  = 100
)

var (
	githubOwnerPattern    = regexp.MustCompile(`^[A-Za-z0-9]+(-[A-Za-z0-9]+)*$`)
	githubRepoNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

type SubscribeRequest struct {
	Email string
	Repo  string
}

func (r *SubscribeRequest) Validate() error {
	if !isValidEmail(r.Email) {
		return apperr.ErrInvalidEmailFormat
	}
	if !isValidGitHubRepoPath(r.Repo) {
		return apperr.ErrInvalidRepositoryFormat
	}
	return nil
}

func isValidEmail(address string) bool {
	if len(address) == 0 || len(address) > maxEmailLength {
		return false
	}

	email, err := mail.ParseAddress(address)
	if err != nil || email.Address != address {
		return false
	}

	at := strings.LastIndex(address, "@")
	if at <= 0 || at == len(address)-1 {
		return false
	}
	return len(address[:at]) <= maxEmailLocalLength
}

func isValidGitHubRepoPath(repo string) bool {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return false
	}
	return isValidGitHubOwner(parts[0]) && isValidGitHubRepoName(parts[1])
}

func isValidGitHubOwner(owner string) bool {
	if len(owner) == 0 || len(owner) > maxGitHubOwnerLength {
		return false
	}
	return githubOwnerPattern.MatchString(owner)
}

func isValidGitHubRepoName(name string) bool {
	if len(name) == 0 || len(name) > maxGitHubRepoLength {
		return false
	}
	if name == "." || name == ".." {
		return false
	}
	if strings.HasSuffix(strings.ToLower(name), ".git") {
		return false
	}

	return githubRepoNamePattern.MatchString(name)
}
