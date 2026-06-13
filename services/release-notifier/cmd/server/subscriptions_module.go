package main

import (
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/eventbus/rabbitmq"
	releasetrackerusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/usecase"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/usecase"
)

func setupSubscriptionsModule(
	log *slog.Logger,
	subscriberRepo *subscriptionpostgresql.SubscriberRepository,
	subscriptionRepo *subscriptionpostgresql.SubscriptionRepository,
	ensureRepositoryTracked *releasetrackerusecase.EnsureRepositoryTracked,
	notificationPublisher *rabbitmq.Publisher,
) subscriptionusecase.SubscriptionUsecases {
	getOrCreateSubscriber := subscriptionusecase.NewGetOrCreateSubscriber(log, subscriberRepo)

	return subscriptionusecase.SubscriptionUsecases{
		SubscribeToRepository: subscriptionusecase.NewSubscribeToRepository(
			log,
			getOrCreateSubscriber,
			ensureRepositoryTracked,
			subscriptionRepo,
			notificationPublisher,
		),
		ConfirmSubscription:       subscriptionusecase.NewConfirmSubscription(subscriptionRepo),
		UnsubscribeFromRepository: subscriptionusecase.NewUnsubscribeFromRepository(subscriptionRepo),
		ListSubscriptions:         subscriptionusecase.NewListSubscriptions(subscriptionRepo),
	}
}
