package main

import (
	"database/sql"
	"log/slog"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/eventbus/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/github"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/config"
	releasetrackerpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/persistence/postgresql"
	releasetrackerworker "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/worker"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/usecase"
)

type applicationModules struct {
	subscriptionUsecases  subscriptionusecase.SubscriptionUsecases
	releaseScheduler      *releasetrackerworker.Scheduler
	notificationPublisher *rabbitmq.Publisher
}

func setupModules(
	log *slog.Logger,
	db *sql.DB,
	cfg *config.Config,
	githubClient *github.Client,
) (*applicationModules, error) {
	subscriberRepo := subscriptionpostgresql.NewSubscriberRepository(db)
	repositoryRepo := releasetrackerpostgresql.NewRepositoryStore(db)
	subscriptionRepo := subscriptionpostgresql.NewSubscriptionRepository(db)

	notificationPublisher, err := rabbitmq.NewNotificationPublisher(rabbitmq.Config{
		URL:      cfg.EventBus.URL,
		Exchange: cfg.EventBus.NotificationExchange,
		Queue:    cfg.EventBus.NotificationQueue,
		DLQ:      cfg.EventBus.NotificationDLQ,
	})
	if err != nil {
		return nil, err
	}

	ensureRepositoryTracked := setupRepositoryTrackingUsecase(
		log,
		repositoryRepo,
		githubClient,
	)
	subscriptionUsecases := setupSubscriptionsModule(
		log,
		subscriberRepo,
		subscriptionRepo,
		ensureRepositoryTracked,
		notificationPublisher,
	)

	activeSubscriptionsByRepository := subscriptionusecase.NewListActiveSubscriptionsByRepository(subscriptionRepo)
	releaseScheduler := setupReleaseTrackerModule(
		log,
		repositoryRepo,
		activeSubscriptionsByRepository,
		notificationPublisher,
		githubClient,
		cfg.Scanner.Interval,
	)

	return &applicationModules{
		subscriptionUsecases:  subscriptionUsecases,
		releaseScheduler:      releaseScheduler,
		notificationPublisher: notificationPublisher,
	}, nil
}
