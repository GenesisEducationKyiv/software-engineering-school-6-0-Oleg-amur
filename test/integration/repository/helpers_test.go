//go:build integration

package repository_test

import "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"

type subscriptionFixture struct {
	subscriber *model.Subscriber
	repository *model.Repository
	token      string
}

func (s *RepositorySuite) createSubscriptionFixture(
	email string,
	repoName string,
	lastSeenTag string,
	token string,
) subscriptionFixture {
	s.T().Helper()

	subscriber, err := s.subscriberRepo.Create(s.ctx, email)
	s.Require().NoError(err, "create subscriber")
	repository, err := s.repositoryRepo.Create(s.ctx, repoName, lastSeenTag)
	s.Require().NoError(err, "create repository")
	fixture := subscriptionFixture{
		subscriber: subscriber,
		repository: repository,
		token:      token,
	}

	err = s.subscriptionRepo.Create(s.ctx, subscriber.ID, repository.ID, token)
	s.Require().NoError(err, "create subscription")

	return fixture
}

func (s *RepositorySuite) createSubscriptionForRepository(
	repository *model.Repository,
	email string,
	token string,
) subscriptionFixture {
	s.T().Helper()

	subscriber, err := s.subscriberRepo.Create(s.ctx, email)
	s.Require().NoError(err, "create subscriber")
	err = s.subscriptionRepo.Create(s.ctx, subscriber.ID, repository.ID, token)
	s.Require().NoError(err, "create subscription")

	return subscriptionFixture{
		subscriber: subscriber,
		repository: repository,
		token:      token,
	}
}

func (s *RepositorySuite) assertSubscriptionJoin(sub *model.Subscription, want subscriptionFixture) {
	s.T().Helper()

	s.Equal(want.subscriber.ID, sub.SubscriberID)
	s.Equal(want.repository.ID, sub.RepositoryID)
	s.Require().NotNil(sub.Subscriber, "subscription subscriber")
	s.Equal(want.subscriber.Email, sub.Subscriber.Email)
	s.Require().NotNil(sub.Repository, "subscription repository")
	s.Equal(want.repository.Name, sub.Repository.Name)
	s.Equal(want.repository.LastSeenTag, sub.Repository.LastSeenTag)
}
