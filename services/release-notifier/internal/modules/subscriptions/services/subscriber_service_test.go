package services

import (
	"context"
	"errors"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/models"
)

func TestSubscriberService_GetOrCreate(t *testing.T) {
	errGet := errors.New("db error")
	errCreate := errors.New("create error")

	tests := []struct {
		name       string
		email      string
		subscriber *models.Subscriber
		getErr     error
		createErr  error
		wantErr    error
		wantCreate bool
	}{
		{
			name:       "returns subscriber when it already exists",
			email:      "test@example.com",
			subscriber: &models.Subscriber{ID: 1, Email: "test@example.com"},
		},
		{
			name:       "creates subscriber when it does not exist",
			email:      "new@example.com",
			subscriber: &models.Subscriber{ID: 2, Email: "new@example.com"},
			getErr:     apperr.ErrNotFound,
			wantCreate: true,
		},
		{
			name:    "returns wrapped get error",
			email:   "error@example.com",
			getErr:  errGet,
			wantErr: errGet,
		},
		{
			name:       "returns wrapped create error",
			email:      "error@example.com",
			getErr:     apperr.ErrNotFound,
			createErr:  errCreate,
			wantErr:    errCreate,
			wantCreate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockSubscriberRepo{
				subscriber: tt.subscriber,
				getErr:     tt.getErr,
				createErr:  tt.createErr,
			}
			svc := NewSubscriberService(testLogger(), repo)

			subscriber, err := svc.GetOrCreate(context.Background(), tt.email)

			assertErrorIs(t, err, tt.wantErr)
			if tt.wantErr == nil && subscriber.ID != tt.subscriber.ID {
				t.Errorf("got subscriber ID %d, want %d", subscriber.ID, tt.subscriber.ID)
			}

			if tt.wantCreate && !repo.createCalled {
				t.Error("want Create to be called")
			}
		})
	}
}

type mockSubscriberRepo struct {
	subscriber   *models.Subscriber
	getErr       error
	createErr    error
	createCalled bool
}

func (f *mockSubscriberRepo) GetByEmail(ctx context.Context, email string) (*models.Subscriber, error) {
	return f.subscriber, f.getErr
}

func (f *mockSubscriberRepo) Create(ctx context.Context, email string) (*models.Subscriber, error) {
	f.createCalled = true
	return f.subscriber, f.createErr
}
