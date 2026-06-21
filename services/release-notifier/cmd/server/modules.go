package main

import (
	"database/sql"
	"log/slog"
	"net/http"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/eventbus/rabbitmq"
	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/releasetracker"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/config"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/usecase"
	subscriptionworkflow "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/workflow"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/worker"
)

type applicationModules struct {
	subscriptionUsecases     subscriptionusecase.SubscriptionUsecases
	subscriptionOutboxRelay  *worker.Scheduler
	subscriptionSaga         *subscriptionworkflow.SubscriptionConfirmationSaga
	subscriptionSagaConsumer *rabbitmq.SubscriptionSagaConsumer
	notificationPublisher    *rabbitmq.Publisher
}

func setupModules(
	log *slog.Logger,
	db *sql.DB,
	cfg *config.Config,
) (*applicationModules, error) {
	queryable := postgresqladapter.NewContextQueryable(db)
	transactionManager := postgresqladapter.NewTransactionManager(db)
	subscriberStore := subscriptionpostgresql.NewSubscriberStore(queryable)
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

	releaseTrackerTimeout, err := time.ParseDuration(cfg.ReleaseTracker.Timeout)
	if err != nil {
		return nil, err
	}
	repositoryTracker := releasetracker.NewClient(
		&http.Client{Timeout: releaseTrackerTimeout},
		cfg.ReleaseTracker.URL,
	)
	subscriptionUsecases := setupSubscriptionsModule(
		log,
		subscriberStore,
		subscriptionStore,
		subscriptionSaga,
		repositoryTracker,
		repositoryTracker,
	)
	publishSubscriptionOutbox := subscriptionworkflow.NewPublishSubscriptionOutbox(
		log,
		subscriptionOutboxStore,
		notificationPublisher,
	)
	subscriptionOutboxRelay := worker.NewScheduler(log, publishSubscriptionOutbox, 5*time.Second)

	return &applicationModules{
		subscriptionUsecases:     subscriptionUsecases,
		subscriptionOutboxRelay:  subscriptionOutboxRelay,
		subscriptionSaga:         subscriptionSaga,
		subscriptionSagaConsumer: subscriptionSagaConsumer,
		notificationPublisher:    notificationPublisher,
	}, nil
}
