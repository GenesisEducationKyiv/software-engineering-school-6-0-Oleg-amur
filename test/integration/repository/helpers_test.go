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
