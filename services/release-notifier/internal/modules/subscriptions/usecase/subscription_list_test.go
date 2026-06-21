package usecase

import (
	"context"
	"database/sql"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

func TestListSubscriptions_Execute(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		subs    []domain.Subscription
		repoErr error
		wantErr error
		wantLen int
	}{
		{
			name:    "returns repository query error",
			email:   "test@example.com",
			repoErr: sql.ErrConnDone,
			wantErr: sql.ErrConnDone,
		},
		{
			name:  "returns empty list when user has no active subscriptions",
			email: "test@example.com",
			subs:  []domain.Subscription{},
		},
		{
			name:  "maps active subscriptions to DTOs",
			email: "test@example.com",
			subs: []domain.Subscription{
				{
					SubscriptionStatus: domain.StatusActive,
					Repository: &domain.Repository{
						Name:        "owner/repo1",
						LastSeenTag: "v1.0",
					},
				},
				{
					SubscriptionStatus: domain.StatusActive,
					Repository: &domain.Repository{
						Name:        "owner/repo2",
						LastSeenTag: "v2.0",
					},
				},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			usecase := NewListSubscriptions(&mockSubscriptionRepo{
				getActiveByEmailSubs: tt.subs,
				getActiveByEmailErr:  tt.repoErr,
			}, &mockRepositoryTracker{})

			subs, err := usecase.Execute(context.Background(), tt.email)

			assertErrorIs(t, err, tt.wantErr)
			if len(subs) != tt.wantLen {
				t.Errorf("got %d subscriptions, want %d", len(subs), tt.wantLen)
			}
		})
	}
}
