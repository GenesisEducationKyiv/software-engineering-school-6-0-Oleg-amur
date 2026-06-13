package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

func TestSubscribeToRepository_Execute(t *testing.T) {
	errSubscriber := errors.New("subscriber service error")

	tests := []struct {
		name          string
		req           SubscribeRequest
		subscriber    *domain.Subscriber
		subscriberErr error
		repository    *domain.RepositoryRef
		repositoryErr error
		createErr     error
		publishErr    error
		wantErr       error
	}{
		{
			name:          "returns subscriber service error",
			req:           SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriberErr: errSubscriber,
			wantErr:       errSubscriber,
		},
		{
			name:          "returns repository service error",
			req:           SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriber:    &domain.Subscriber{ID: 1},
			repositoryErr: apperr.ErrRepoNotFound,
			wantErr:       apperr.ErrRepoNotFound,
		},
		{
			name:       "returns already subscribed when repository rejects duplicate",
			req:        SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriber: &domain.Subscriber{ID: 1},
			repository: &domain.RepositoryRef{ID: 1},
			createErr:  apperr.ErrAlreadyExists,
			wantErr:    apperr.ErrAlreadySubscribed,
		},
		{
			name:       "enqueues confirmation event after successful subscription",
			req:        SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriber: &domain.Subscriber{ID: 1},
			repository: &domain.RepositoryRef{ID: 1},
		},
		{
			name:       "returns publish error after successful subscription",
			req:        SubscribeRequest{Email: "test@example.com", Repo: "owner/repo"},
			subscriber: &domain.Subscriber{ID: 1},
			repository: &domain.RepositoryRef{ID: 1},
			publishErr: errors.New("broker down"),
			wantErr:    errors.New("broker down"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			publisher := &mockNotificationPublisher{confirmationErr: tt.publishErr}
			usecase := NewSubscribeToRepository(
				testLogger(),
				&mockSubscriberRegistration{subscriber: tt.subscriber, err: tt.subscriberErr},
				&mockRepositoryService{repository: tt.repository, err: tt.repositoryErr},
				&mockSubscriptionRepo{createErr: tt.createErr},
				publisher,
			)

			err := usecase.Execute(context.Background(), tt.req)

			if tt.wantErr == nil {
				assertErrorIs(t, err, nil)
				assertSubscriptionEvent(t, publisher, tt.req.Email)
			} else if err == nil {
				t.Fatalf("got nil error, want %v", tt.wantErr)
			}
		})
	}
}
