package model

import (
	"errors"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
)

func TestSubscribeRequest_Validate(t *testing.T) {
	tests := []struct {
		name    string
		request SubscribeRequest
		wantErr error
	}{
		{
			name: "accepts non-empty email and owner repo address",
			request: SubscribeRequest{
				Email: "user@example.com",
				Repo:  "owner/repo",
			},
		},
		{
			name: "returns invalid format when email is empty",
			request: SubscribeRequest{
				Repo: "owner/repo",
			},
			wantErr: apperr.ErrInvalidFormat,
		},
		{
			name: "returns invalid format when repo is empty",
			request: SubscribeRequest{
				Email: "user@example.com",
			},
			wantErr: apperr.ErrInvalidFormat,
		},
		{
			name: "returns invalid format when repo has no slash",
			request: SubscribeRequest{
				Email: "user@example.com",
				Repo:  "owner",
			},
			wantErr: apperr.ErrInvalidFormat,
		},
		{
			name: "returns invalid format when owner is empty",
			request: SubscribeRequest{
				Email: "user@example.com",
				Repo:  "/repo",
			},
			wantErr: apperr.ErrInvalidFormat,
		},
		{
			name: "returns invalid format when repo name is empty",
			request: SubscribeRequest{
				Email: "user@example.com",
				Repo:  "owner/",
			},
			wantErr: apperr.ErrInvalidFormat,
		},
		{
			name: "returns invalid format when repo has extra path parts",
			request: SubscribeRequest{
				Email: "user@example.com",
				Repo:  "owner/repo/extra",
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

func assertErrorIs(t *testing.T, got error, want error) {
	t.Helper()

	if !errors.Is(got, want) {
		t.Fatalf("got error %v, want %v", got, want)
	}
}
