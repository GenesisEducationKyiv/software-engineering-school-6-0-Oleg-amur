//go:build integration

package repository_test

import subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"

type createdSubscription struct {
	subscriber   *subscriptionmodels.Subscriber
	repositoryID int64
	token        string
}

func (s *RepositorySuite) createSubscription(
	email string,
	repositoryID int64,
	token string,
) createdSubscription {
	s.T().Helper()

	subscriber, err := s.subscriberStore.Create(s.ctx, email)
	s.Require().NoError(err, "create subscriber")
	subscription := createdSubscription{
		subscriber:   subscriber,
		repositoryID: repositoryID,
		token:        token,
	}

	_, err = s.subscriptionStore.Create(s.ctx, subscriber.ID, repositoryID, token)
	s.Require().NoError(err, "create subscription")

	return subscription
}
