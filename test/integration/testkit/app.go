//go:build integration

package testkit

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/repository/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/service"
)

func (s *Suite) NewSubscriptionService() (
	*service.SubscriptionService,
	*FakeGithubClient,
	chan model.SubscriptionEvent,
) {
	subscriberRepo := postgresql.NewSubscriberRepository(s.DB)
	repositoryRepo := postgresql.NewRepositoryRepository(s.DB)
	subscriptionRepo := postgresql.NewSubscriptionRepository(s.DB)
	githubClient := NewFakeGithubClient()

	subscriberService := service.NewSubscriberService(s.Logger, subscriberRepo)
	repositoryService := service.NewRepositoryService(s.Logger, repositoryRepo, githubClient)
	subscriptionEvents := make(chan model.SubscriptionEvent, 10)
	subscriptionService := service.NewSubscriptionService(
		s.Logger,
		subscriberService,
		repositoryService,
		subscriptionRepo,
		subscriptionEvents,
	)

	return subscriptionService, githubClient, subscriptionEvents
}
