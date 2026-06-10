//go:build integration

package repository_test

import (
	releasewatchmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasewatch/models"
	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/models"
)

type createdSubscription struct {
	subscriber *subscriptionmodels.Subscriber
	repository *releasewatchmodels.TrackedRepository
	token      string
}

func (s *RepositorySuite) createSubscription(
	email string,
	repoName string,
	lastSeenTag string,
	token string,
) createdSubscription {
	s.T().Helper()

	subscriber, err := s.subscriberRepo.Create(s.ctx, email)
	s.Require().NoError(err, "create subscriber")
	repository, err := s.repositoryRepo.Create(s.ctx, repoName, lastSeenTag)
	s.Require().NoError(err, "create repository")
	subscription := createdSubscription{
		subscriber: subscriber,
		repository: repository,
		token:      token,
	}

	err = s.subscriptionRepo.Create(s.ctx, subscriber.ID, repository.ID, token)
	s.Require().NoError(err, "create subscription")

	return subscription
}
