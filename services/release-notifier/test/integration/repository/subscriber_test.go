//go:build integration

package repository_test

import "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"

func (s *RepositorySuite) TestSubscriberStore_CreateAndGetByEmail() {
	created, err := s.subscriberStore.Create(s.ctx, "user@example.com")
	s.Require().NoError(err, "create subscriber")
	s.NotZero(created.ID, "created subscriber id")
	s.Equal("user@example.com", created.Email)

	got, err := s.subscriberStore.GetByEmail(s.ctx, "user@example.com")
	s.Require().NoError(err, "get subscriber by email")
	s.Equal(created.ID, got.ID)
}

func (s *RepositorySuite) TestSubscriberStore_GetByEmailReturnsNotFound() {
	_, err := s.subscriberStore.GetByEmail(s.ctx, "missing@example.com")

	s.ErrorIs(err, apperr.ErrNotFound)
}
