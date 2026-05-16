//go:build integration

package repository_test

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
)

func (s *RepositorySuite) TestSubscriptionRepository_CreateRejectsDuplicateSubscriberRepository() {
	fixture := s.createSubscriptionFixture("user@example.com", "owner/repo", "v1.0.0", "token-1")

	err := s.subscriptionRepo.Create(s.ctx, fixture.subscriber.ID, fixture.repository.ID, "token-2")

	s.ErrorIs(err, apperr.ErrAlreadyExists)
}

func (s *RepositorySuite) TestSubscriptionRepository_GetByTokenReturnsJoinedData() {
	fixture := s.createSubscriptionFixture("user@example.com", "owner/repo", "v1.0.0", "token-1")

	got, err := s.subscriptionRepo.GetByToken(s.ctx, fixture.token)
	s.Require().NoError(err, "get subscription by token")

	s.assertSubscriptionJoin(got, fixture)
	s.Equal(model.StatusPending, got.SubscriptionStatus)
}

func (s *RepositorySuite) TestSubscriptionRepository_GetByTokenReturnsNotFound() {
	_, err := s.subscriptionRepo.GetByToken(s.ctx, "missing-token")

	s.ErrorIs(err, apperr.ErrNotFound)
}

func (s *RepositorySuite) TestSubscriptionRepository_ActivateMakesSubscriptionVisibleByEmail() {
	fixture := s.createSubscriptionFixture("user@example.com", "owner/repo", "v1.0.0", "token-1")

	beforeActivation, err := s.subscriptionRepo.GetActiveByEmail(s.ctx, fixture.subscriber.Email)
	s.Require().NoError(err, "get active subscriptions before activation")
	s.Empty(beforeActivation, "active subscriptions before activation")

	err = s.subscriptionRepo.Activate(s.ctx, fixture.token)
	s.Require().NoError(err, "activate subscription")

	active, err := s.subscriptionRepo.GetActiveByEmail(s.ctx, fixture.subscriber.Email)
	s.Require().NoError(err, "get active subscriptions by email")
	s.Require().Len(active, 1, "active subscriptions by email")
	s.Equal(fixture.repository.Name, active[0].Repository.Name)
	s.Equal(fixture.repository.LastSeenTag, active[0].Repository.LastSeenTag)
}

func (s *RepositorySuite) TestSubscriptionRepository_ActivateReturnsNotFound() {
	err := s.subscriptionRepo.Activate(s.ctx, "missing-token")

	s.ErrorIs(err, apperr.ErrNotFound)
}

func (s *RepositorySuite) TestSubscriptionRepository_GetActiveByRepoIDReturnsOnlyActiveSubscribers() {
	activeFixture := s.createSubscriptionFixture("active@example.com", "owner/repo", "v1.0.0", "active-token")
	pendingFixture := s.createSubscriptionForRepository(activeFixture.repository, "pending@example.com", "pending-token")

	err := s.subscriptionRepo.Activate(s.ctx, activeFixture.token)
	s.Require().NoError(err, "activate subscription")

	active, err := s.subscriptionRepo.GetActiveByRepoID(s.ctx, activeFixture.repository.ID)
	s.Require().NoError(err, "get active subscriptions by repository id")
	s.Require().Len(active, 1, "active subscriptions by repository id")
	s.Equal(activeFixture.subscriber.Email, active[0].Subscriber.Email)
	s.NotEqual(pendingFixture.subscriber.Email, active[0].Subscriber.Email)
}

func (s *RepositorySuite) TestSubscriptionRepository_DeleteByTokenRemovesSubscription() {
	fixture := s.createSubscriptionFixture("user@example.com", "owner/repo", "v1.0.0", "token-1")

	err := s.subscriptionRepo.DeleteByToken(s.ctx, fixture.token)
	s.Require().NoError(err, "delete subscription by token")
	_, err = s.subscriptionRepo.GetByToken(s.ctx, fixture.token)
	s.ErrorIs(err, apperr.ErrNotFound)
}
