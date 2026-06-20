//go:build integration

package repository_test

import (
	releasetrackerdomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/domain"
	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
)

type createdSubscription struct {
	subscriber *subscriptionmodels.Subscriber
	repository *releasetrackerdomain.Repository
	token      string
}

func (s *RepositorySuite) createSubscription(
	email string,
	repoName string,
	lastSeenTag string,
	token string,
) createdSubscription {
	s.T().Helper()

	subscriber, err := s.subscriberStore.Create(s.ctx, email)
	s.Require().NoError(err, "create subscriber")
	repository, err := s.repositoryStore.Create(s.ctx, repoName, lastSeenTag)
	s.Require().NoError(err, "create repository")
	subscription := createdSubscription{
		subscriber: subscriber,
		repository: repository,
		token:      token,
	}

	_, err = s.subscriptionStore.Create(s.ctx, subscriber.ID, repository.ID, token)
	s.Require().NoError(err, "create subscription")

	return subscription
}
