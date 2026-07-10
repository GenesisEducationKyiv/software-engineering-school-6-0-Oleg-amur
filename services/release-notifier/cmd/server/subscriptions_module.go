package main

import (
	"log/slog"

	releasetrackerusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/usecase"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/usecase"
)

func setupSubscriptionsModule(
	log *slog.Logger,
	subscriberStore *subscriptionpostgresql.SubscriberStore,
	subscriptionStore *subscriptionpostgresql.SubscriptionStore,
	subscriptionCreator subscriptionusecase.SubscriptionCreator,
	ensureRepositoryTracked *releasetrackerusecase.EnsureRepositoryTracked,
) subscriptionusecase.SubscriptionUsecases {
	getOrCreateSubscriber := subscriptionusecase.NewGetOrCreateSubscriber(log, subscriberStore)

	return subscriptionusecase.SubscriptionUsecases{
		SubscribeToRepository: subscriptionusecase.NewSubscribeToRepository(
			log,
			getOrCreateSubscriber,
			ensureRepositoryTracked,
			subscriptionCreator,
		),
		ConfirmSubscription:       subscriptionusecase.NewConfirmSubscription(subscriptionStore),
		UnsubscribeFromRepository: subscriptionusecase.NewUnsubscribeFromRepository(subscriptionStore),
		ListSubscriptions:         subscriptionusecase.NewListSubscriptions(subscriptionStore),
	}
}
