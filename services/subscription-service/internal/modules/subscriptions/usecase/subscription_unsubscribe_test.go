package usecase

import (
	"context"
	"database/sql"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/apperr"
)

func TestUnsubscribeFromRepository_Execute(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		deleteErr error
		wantErr   error
	}{
		{
			name:      "returns repository delete error",
			token:     "test-token",
			deleteErr: sql.ErrConnDone,
			wantErr:   sql.ErrConnDone,
		},
		{
			name:      "returns token not found when repository has no token",
			token:     "test-token",
			deleteErr: apperr.ErrNotFound,
			wantErr:   apperr.ErrTokenNotFound,
		},
		{
			name:  "deletes subscription by token",
			token: "test-token",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usecase := NewUnsubscribeFromRepository(&mockSubscriptionRepo{deleteErr: tt.deleteErr})

			err := usecase.Execute(context.Background(), tt.token)

			assertErrorIs(t, err, tt.wantErr)
		})
	}
}
