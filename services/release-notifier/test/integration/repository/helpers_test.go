//go:build integration

package repository_test

import subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"

type createdSubscription struct {
	subscriber     *subscriptionmodels.Subscriber
	repositoryName string
	token          string
}

func (s *RepositorySuite) createSubscription(
	email string,
	repoName string,
	_ string,
	token string,
) createdSubscription {
	s.T().Helper()

	subscriber, err := s.subscriberStore.Create(s.ctx, email)
	s.Require().NoError(err, "create subscriber")
	subscription := createdSubscription{
		subscriber:     subscriber,
		repositoryName: repoName,
		token:          token,
	}

	_, err = s.subscriptionStore.Create(s.ctx, subscriber.ID, repoName, token)
	s.Require().NoError(err, "create subscription")

	return subscription
}
