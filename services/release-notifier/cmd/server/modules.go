package main

import (
	"database/sql"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/eventbus/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/github"
	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/config"
	releasetrackerpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/persistence/postgresql"
	releasetrackerworker "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/worker"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/usecase"
	subscriptionworkflow "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/workflow"
)

type applicationModules struct {
	subscriptionUsecases     subscriptionusecase.SubscriptionUsecases
	releaseScheduler         *releasetrackerworker.Scheduler
	subscriptionOutboxRelay  *releasetrackerworker.Scheduler
	subscriptionSaga         *subscriptionworkflow.SubscriptionConfirmationSaga
	subscriptionSagaConsumer *rabbitmq.SubscriptionSagaConsumer
	notificationPublisher    *rabbitmq.Publisher
}

func setupModules(
	log *slog.Logger,
	db *sql.DB,
	cfg *config.Config,
	githubClient *github.Client,
) (*applicationModules, error) {
	queryable := postgresqladapter.NewContextQueryable(db)
	transactionManager := postgresqladapter.NewTransactionManager(db)
	subscriberStore := subscriptionpostgresql.NewSubscriberStore(queryable)
	repositoryStore := releasetrackerpostgresql.NewRepositoryStore(queryable)
	subscriptionStore := subscriptionpostgresql.NewSubscriptionStore(queryable)
	subscriptionSagaStore := subscriptionpostgresql.NewSubscriptionSagaStore(queryable)
	subscriptionOutboxStore := subscriptionpostgresql.NewOutboxStore(queryable)
	subscriptionSaga := subscriptionworkflow.NewSubscriptionConfirmationSaga(
		log,
		transactionManager,
		subscriptionStore,
		subscriptionSagaStore,
		subscriptionOutboxStore,
	)

	notificationPublisher, err := rabbitmq.NewNotificationPublisher(rabbitmq.Config{
		URL:      cfg.EventBus.URL,
		Exchange: cfg.EventBus.NotificationExchange,
		Queue:    cfg.EventBus.NotificationQueue,
		DLQ:      cfg.EventBus.NotificationDLQ,
	})
	if err != nil {
		return nil, err
	}

	subscriptionSagaConsumer, err := rabbitmq.NewSubscriptionSagaConsumer(log, rabbitmq.Config{
		URL:      cfg.EventBus.URL,
		Exchange: cfg.EventBus.NotificationExchange,
		Queue:    cfg.EventBus.SubscriptionSagaQueue,
		DLQ:      cfg.EventBus.SubscriptionSagaDLQ,
	})
	if err != nil {
		_ = notificationPublisher.Close()
		return nil, err
	}

	ensureRepositoryTracked := setupRepositoryTrackingUsecase(
		log,
		repositoryStore,
		githubClient,
	)
	subscriptionUsecases := setupSubscriptionsModule(
		log,
		subscriberStore,
		subscriptionStore,
		subscriptionSaga,
		ensureRepositoryTracked,
	)

	activeSubscriptionsByRepository := subscriptionusecase.NewListActiveSubscriptionsByRepository(subscriptionStore)
	releaseScheduler := setupReleaseTrackerModule(
		log,
		repositoryStore,
		activeSubscriptionsByRepository,
		notificationPublisher,
		githubClient,
		cfg.Scanner.Interval,
	)
	publishSubscriptionOutbox := subscriptionworkflow.NewPublishSubscriptionOutbox(
		log,
		subscriptionOutboxStore,
		notificationPublisher,
	)
	subscriptionOutboxRelay := releasetrackerworker.NewScheduler(log, publishSubscriptionOutbox, 5*time.Second)

	return &applicationModules{
		subscriptionUsecases:     subscriptionUsecases,
		releaseScheduler:         releaseScheduler,
		subscriptionOutboxRelay:  subscriptionOutboxRelay,
		subscriptionSaga:         subscriptionSaga,
		subscriptionSagaConsumer: subscriptionSagaConsumer,
		notificationPublisher:    notificationPublisher,
	}, nil
}
