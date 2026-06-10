package service

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
)

func TestSubscriberService_GetOrCreate(t *testing.T) {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	tests := []struct {
		name          string
		email         string
		mockSub       *model.Subscriber
		getErr        error
		createErr     error
		expectedError error
		expectCreate  bool
	}{
		{
			name:    "Subscriber exists",
			email:   "test@example.com",
			mockSub: &model.Subscriber{ID: 1, Email: "test@example.com"},
		},
		{
			name:         "Subscriber created",
			email:        "new@example.com",
			mockSub:      &model.Subscriber{ID: 2, Email: "new@example.com"},
			getErr:       apperr.ErrNotFound,
			expectCreate: true,
		},
		{
			name:          "DB error on get",
			email:         "error@example.com",
			getErr:        errors.New("db error"),
			expectedError: errors.New("subscriber check error: db error"),
		},
		{
			name:          "DB error on create",
			email:         "error@example.com",
			getErr:        apperr.ErrNotFound,
			createErr:     errors.New("create error"),
			expectedError: errors.New("subscriber create error: create error"),
			expectCreate:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockSubscriberRepoForService{
				sub:       tt.mockSub,
				getErr:    tt.getErr,
				createErr: tt.createErr,
			}
			s := NewSubscriberService(log, repo)

			sub, err := s.GetOrCreate(context.Background(), tt.email)

			if tt.expectedError != nil {
				if err == nil || err.Error() != tt.expectedError.Error() {
					t.Errorf("expected error %v, got %v", tt.expectedError, err)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if sub.ID != tt.mockSub.ID {
					t.Errorf("expected ID %d, got %d", tt.mockSub.ID, sub.ID)
				}
			}

			if tt.expectCreate && !repo.createCalled {
				t.Error("expected Create to be called")
			}
		})
	}
}

type mockSubscriberRepoForService struct {
	sub          *model.Subscriber
	getErr       error
	createErr    error
	createCalled bool
}

func (m *mockSubscriberRepoForService) GetByEmail(ctx context.Context, email string) (*model.Subscriber, error) {
	return m.sub, m.getErr
}

func (m *mockSubscriberRepoForService) Create(ctx context.Context, email string) (*model.Subscriber, error) {
	m.createCalled = true
	return m.sub, m.createErr
}
