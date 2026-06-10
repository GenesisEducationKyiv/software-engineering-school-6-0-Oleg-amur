//go:build integration

package repository_test

import "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"

func (s *RepositorySuite) TestSubscriberRepository_CreateAndGetByEmail() {
	created, err := s.subscriberRepo.Create(s.ctx, "user@example.com")
	s.Require().NoError(err, "create subscriber")
	s.NotZero(created.ID, "created subscriber id")
	s.Equal("user@example.com", created.Email)

	got, err := s.subscriberRepo.GetByEmail(s.ctx, "user@example.com")
	s.Require().NoError(err, "get subscriber by email")
	s.Equal(created.ID, got.ID)
}

func (s *RepositorySuite) TestSubscriberRepository_GetByEmailReturnsNotFound() {
	_, err := s.subscriberRepo.GetByEmail(s.ctx, "missing@example.com")

	s.ErrorIs(err, apperr.ErrNotFound)
}
