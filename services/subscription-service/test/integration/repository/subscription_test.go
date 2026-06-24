//go:build integration

package repository_test

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/apperr"
	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
)

func (s *RepositorySuite) TestSubscriptionStore_CreateRejectsDuplicateSubscriberRepository() {
	subscription := s.createSubscription("user@example.com", "owner/repo", "v1.0.0", "token-1")

	_, err := s.subscriptionStore.Create(s.ctx, subscription.subscriber.ID, subscription.repositoryName, "token-2")

	s.ErrorIs(err, apperr.ErrAlreadyExists)
}

func (s *RepositorySuite) TestSubscriptionStore_GetByTokenReturnsJoinedData() {
	subscription := s.createSubscription("user@example.com", "owner/repo", "v1.0.0", "token-1")

	got, err := s.subscriptionStore.GetByToken(s.ctx, subscription.token)
	s.Require().NoError(err, "get subscription by token")

	s.Equal(subscription.subscriber.ID, got.SubscriberID)
	s.Equal(subscription.repositoryName, got.RepositoryName)
	s.Require().NotNil(got.Subscriber, "subscription subscriber")
	s.Equal(subscription.subscriber.Email, got.Subscriber.Email)
	s.Require().NotNil(got.Repository, "subscription repository")
	s.Equal(subscription.repositoryName, got.Repository.Name)
	s.Equal(subscriptionmodels.StatusPending, got.SubscriptionStatus)
}

func (s *RepositorySuite) TestSubscriptionStore_GetByTokenReturnsNotFound() {
	_, err := s.subscriptionStore.GetByToken(s.ctx, "missing-token")

	s.ErrorIs(err, apperr.ErrNotFound)
}

func (s *RepositorySuite) TestSubscriptionStore_ActivateMakesSubscriptionVisibleByEmail() {
	subscription := s.createSubscription("user@example.com", "owner/repo", "v1.0.0", "token-1")

	beforeActivation, err := s.subscriptionStore.GetActiveByEmail(s.ctx, subscription.subscriber.Email)
	s.Require().NoError(err, "get active subscriptions before activation")
	s.Empty(beforeActivation, "active subscriptions before activation")

	err = s.subscriptionStore.Activate(s.ctx, subscription.token)
	s.Require().NoError(err, "activate subscription")

	active, err := s.subscriptionStore.GetActiveByEmail(s.ctx, subscription.subscriber.Email)
	s.Require().NoError(err, "get active subscriptions by email")
	s.Require().Len(active, 1, "active subscriptions by email")
	s.Equal(subscription.repositoryName, active[0].Repository.Name)
}

func (s *RepositorySuite) TestSubscriptionStore_ActivateReturnsNotFound() {
	err := s.subscriptionStore.Activate(s.ctx, "missing-token")

	s.ErrorIs(err, apperr.ErrNotFound)
}

func (s *RepositorySuite) TestSubscriptionStore_GetActiveByRepoNameReturnsOnlyActiveSubscribers() {
	activeSubscription := s.createSubscription("active@example.com", "owner/repo", "v1.0.0", "active-token")
	pendingSubscriber, err := s.subscriberStore.Create(s.ctx, "pending@example.com")
	s.Require().NoError(err, "create pending subscriber")
	_, err = s.subscriptionStore.Create(s.ctx, pendingSubscriber.ID, activeSubscription.repositoryName, "pending-token")
	s.Require().NoError(err, "create pending subscription")

	err = s.subscriptionStore.Activate(s.ctx, activeSubscription.token)
	s.Require().NoError(err, "activate subscription")

	active, err := s.subscriptionStore.GetActiveByRepositoryName(s.ctx, activeSubscription.repositoryName)
	s.Require().NoError(err, "get active subscriptions by repository name")
	s.Require().Len(active, 1, "active subscriptions by repository name")
	s.Equal(activeSubscription.subscriber.Email, active[0].Email)
	s.NotEqual(pendingSubscriber.Email, active[0].Email)
}

func (s *RepositorySuite) TestSubscriptionStore_DeleteByTokenRemovesSubscription() {
	subscription := s.createSubscription("user@example.com", "owner/repo", "v1.0.0", "token-1")

	err := s.subscriptionStore.DeleteByToken(s.ctx, subscription.token)
	s.Require().NoError(err, "delete subscription by token")
	_, err = s.subscriptionStore.GetByToken(s.ctx, subscription.token)
	s.ErrorIs(err, apperr.ErrNotFound)
}
