package dto

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
)

type SubscribeRequest struct {
	Email string
	Repo  string
}

func (r *SubscribeRequest) Validate() error {
	if !isValidEmail(r.Email) {
		return apperr.ErrInvalidFormat
	}
	if !isValidGitHubRepoPath(r.Repo) {
		return apperr.ErrInvalidFormat
	}
	return nil
}
