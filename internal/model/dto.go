package model

import (
	"net/mail"
	"strings"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
)

type SubscribeRequest struct {
	Email string
	Repo  string
}

func (r *SubscribeRequest) Validate() error {
	if r.Email == "" {
		return apperr.ErrInvalidFormat
	}
	email, err := mail.ParseAddress(r.Email)
	if err != nil || email.Address != r.Email {
		return apperr.ErrInvalidFormat
	}
	parts := strings.Split(r.Repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return apperr.ErrInvalidFormat
	}
	return nil
}

type SubscriptionDTO struct {
	Email       string
	Repo        string
	Confirmed   bool
	LastSeenTag string
}
