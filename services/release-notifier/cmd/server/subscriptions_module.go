package main

import (
	"log/slog"

	releasetrackerusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/usecase"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/usecase"
)

func setupSubscriptionsModule(
	log *slog.Logger,
	subscriberRepo *subscriptionpostgresql.SubscriberRepository,
	subscriptionRepo *subscriptionpostgresql.SubscriptionRepository,
	sagaStore *subscriptionpostgresql.SagaStore,
	ensureRepositoryTracked *releasetrackerusecase.EnsureRepositoryTracked,
) subscriptionusecase.SubscriptionUsecases {
	getOrCreateSubscriber := subscriptionusecase.NewGetOrCreateSubscriber(log, subscriberRepo)

	return subscriptionusecase.SubscriptionUsecases{
		SubscribeToRepository: subscriptionusecase.NewSubscribeToRepository(
			log,
			getOrCreateSubscriber,
			ensureRepositoryTracked,
			sagaStore,
		),
		ConfirmSubscription:       subscriptionusecase.NewConfirmSubscription(subscriptionRepo),
		UnsubscribeFromRepository: subscriptionusecase.NewUnsubscribeFromRepository(subscriptionRepo),
		ListSubscriptions:         subscriptionusecase.NewListSubscriptions(subscriptionRepo),
	}
}
