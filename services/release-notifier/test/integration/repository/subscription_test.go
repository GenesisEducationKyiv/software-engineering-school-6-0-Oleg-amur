//go:build integration

package repository_test

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

func (s *RepositorySuite) TestSubscriptionRepository_CreateRejectsDuplicateSubscriberRepository() {
	subscription := s.createSubscription("user@example.com", "owner/repo", "v1.0.0", "token-1")

	err := s.subscriptionRepo.Create(s.ctx, subscription.subscriber.ID, subscription.repository.ID, "token-2")

	s.ErrorIs(err, apperr.ErrAlreadyExists)
}

func (s *RepositorySuite) TestSubscriptionRepository_GetByTokenReturnsJoinedData() {
	subscription := s.createSubscription("user@example.com", "owner/repo", "v1.0.0", "token-1")

	got, err := s.subscriptionRepo.GetByToken(s.ctx, subscription.token)
	s.Require().NoError(err, "get subscription by token")

	s.Equal(subscription.subscriber.ID, got.SubscriberID)
	s.Equal(subscription.repository.ID, got.RepositoryID)
	s.Require().NotNil(got.Subscriber, "subscription subscriber")
	s.Equal(subscription.subscriber.Email, got.Subscriber.Email)
	s.Require().NotNil(got.Repository, "subscription repository")
	s.Equal(subscription.repository.Name, got.Repository.Name)
	s.Equal(subscription.repository.LastSeenTag, got.Repository.LastSeenTag)
	s.Equal(subscriptionmodels.StatusPending, got.SubscriptionStatus)
}

func (s *RepositorySuite) TestSubscriptionRepository_GetByTokenReturnsNotFound() {
	_, err := s.subscriptionRepo.GetByToken(s.ctx, "missing-token")

	s.ErrorIs(err, apperr.ErrNotFound)
}

func (s *RepositorySuite) TestSubscriptionRepository_ActivateMakesSubscriptionVisibleByEmail() {
	subscription := s.createSubscription("user@example.com", "owner/repo", "v1.0.0", "token-1")

	beforeActivation, err := s.subscriptionRepo.GetActiveByEmail(s.ctx, subscription.subscriber.Email)
	s.Require().NoError(err, "get active subscriptions before activation")
	s.Empty(beforeActivation, "active subscriptions before activation")

	err = s.subscriptionRepo.Activate(s.ctx, subscription.token)
	s.Require().NoError(err, "activate subscription")

	active, err := s.subscriptionRepo.GetActiveByEmail(s.ctx, subscription.subscriber.Email)
	s.Require().NoError(err, "get active subscriptions by email")
	s.Require().Len(active, 1, "active subscriptions by email")
	s.Equal(subscription.repository.Name, active[0].Repository.Name)
	s.Equal(subscription.repository.LastSeenTag, active[0].Repository.LastSeenTag)
}

func (s *RepositorySuite) TestSubscriptionRepository_ActivateReturnsNotFound() {
	err := s.subscriptionRepo.Activate(s.ctx, "missing-token")

	s.ErrorIs(err, apperr.ErrNotFound)
}

func (s *RepositorySuite) TestSubscriptionRepository_GetActiveByRepoIDReturnsOnlyActiveSubscribers() {
	activeSubscription := s.createSubscription("active@example.com", "owner/repo", "v1.0.0", "active-token")
	pendingSubscriber, err := s.subscriberRepo.Create(s.ctx, "pending@example.com")
	s.Require().NoError(err, "create pending subscriber")
	err = s.subscriptionRepo.Create(s.ctx, pendingSubscriber.ID, activeSubscription.repository.ID, "pending-token")
	s.Require().NoError(err, "create pending subscription")

	err = s.subscriptionRepo.Activate(s.ctx, activeSubscription.token)
	s.Require().NoError(err, "activate subscription")

	active, err := s.subscriptionRepo.GetActiveByRepositoryID(s.ctx, activeSubscription.repository.ID)
	s.Require().NoError(err, "get active subscriptions by repository id")
	s.Require().Len(active, 1, "active subscriptions by repository id")
	s.Equal(activeSubscription.subscriber.Email, active[0].Email)
	s.NotEqual(pendingSubscriber.Email, active[0].Email)
}

func (s *RepositorySuite) TestSubscriptionRepository_DeleteByTokenRemovesSubscription() {
	subscription := s.createSubscription("user@example.com", "owner/repo", "v1.0.0", "token-1")

	err := s.subscriptionRepo.DeleteByToken(s.ctx, subscription.token)
	s.Require().NoError(err, "delete subscription by token")
	_, err = s.subscriptionRepo.GetByToken(s.ctx, subscription.token)
	s.ErrorIs(err, apperr.ErrNotFound)
}
