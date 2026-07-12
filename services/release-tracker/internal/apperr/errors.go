package apperr

import "errors"

var (
	ErrRepositoryNotFound = errors.New("repository not found")
	ErrRateLimitExceeded  = errors.New("rate limit exceeded")
)
