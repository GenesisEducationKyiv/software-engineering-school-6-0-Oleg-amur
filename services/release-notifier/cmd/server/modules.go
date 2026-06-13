package main

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/eventbus/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/github"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/config"
	releasetrackerpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/persistence/postgresql"
	releasetrackerusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/usecase"
	releasetrackerworker "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/worker"
	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/usecase"
)

type applicationModules struct {
	subscriptionUsecases  subscriptionusecase.SubscriptionUsecases
	releaseScheduler      *releasetrackerworker.Scheduler
	notificationPublisher *rabbitmq.Publisher
}

type repositoryTracker struct {
	service *releasetrackerusecase.RepositoryService
}

func (t repositoryTracker) EnsureTracked(
	ctx context.Context,
	repoName string,
) (*subscriptionmodels.RepositoryRef, error) {
	repo, err := t.service.EnsureTracked(ctx, repoName)
	if err != nil {
		return nil, err
	}

	return &subscriptionmodels.RepositoryRef{
		ID:          repo.ID,
		Name:        repo.Name,
		LastSeenTag: repo.LastSeenTag,
	}, nil
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

	getOrCreateSubscriber := subscriptionusecase.NewGetOrCreateSubscriber(log, subscriberRepo)
	repositoryService := releasetrackerusecase.NewRepositoryService(log, repositoryRepo, githubClient)

	subscriptionUsecases := subscriptionusecase.SubscriptionUsecases{
		SubscribeToRepository: subscriptionusecase.NewSubscribeToRepository(
			log,
			getOrCreateSubscriber,
			repositoryTracker{service: repositoryService},
			subscriptionRepo,
			notificationPublisher,
		),
		ConfirmSubscription:       subscriptionusecase.NewConfirmSubscription(subscriptionRepo),
		UnsubscribeFromRepository: subscriptionusecase.NewUnsubscribeFromRepository(subscriptionRepo),
		ListSubscriptions:         subscriptionusecase.NewListSubscriptions(subscriptionRepo),
	}

	releaseNotificationPlanner := releasetrackerusecase.NewReleaseNotificationPlanner(
		log,
		subscriptionRepo,
		notificationPublisher,
	)
	releaseScanner := releasetrackerusecase.NewReleaseScanner(
		log,
		repositoryRepo,
		githubClient,
		releaseNotificationPlanner,
	)

	scanInterval, err := time.ParseDuration(cfg.Scanner.Interval)
	if err != nil {
		log.Error("failed to parse scanner interval", "val", cfg.Scanner.Interval, "err", err)
		scanInterval = time.Hour
	}

	return &applicationModules{
		subscriptionUsecases:  subscriptionUsecases,
		releaseScheduler:      releasetrackerworker.NewScheduler(log, releaseScanner, scanInterval),
		notificationPublisher: notificationPublisher,
	}, nil
}
