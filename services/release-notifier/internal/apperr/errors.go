package apperr

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrAlreadyExists     = errors.New("already exists")
	ErrInvalidInput      = errors.New("invalid input")
	ErrRateLimitExceeded = errors.New("rate limit exceeded")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrInternal          = errors.New("internal error")
	ErrRepoNotFound      = errors.New("repository not found")
	ErrTokenNotFound     = errors.New("token not found")
	ErrInvalidFormat     = errors.New("invalid repository format")
	ErrAlreadySubscribed = errors.New("already subscribed to this repository")
)
