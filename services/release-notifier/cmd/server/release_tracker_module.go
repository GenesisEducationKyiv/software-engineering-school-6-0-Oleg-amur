package main

import (
	"log/slog"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/eventbus/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/github"
	releasetrackerpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/persistence/postgresql"
	releasetrackerusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/usecase"
	releasetrackerworker "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/worker"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
)

func setupRepositoryTrackingUsecase(
	log *slog.Logger,
	repositoryRepo *releasetrackerpostgresql.RepositoryStore,
	githubClient *github.Client,
) *releasetrackerusecase.EnsureRepositoryTracked {
	return releasetrackerusecase.NewEnsureRepositoryTracked(log, repositoryRepo, githubClient)
}

func setupReleaseTrackerModule(
	log *slog.Logger,
	repositoryRepo *releasetrackerpostgresql.RepositoryStore,
	subscriptionRepo *subscriptionpostgresql.SubscriptionRepository,
	notificationPublisher *rabbitmq.Publisher,
	githubClient *github.Client,
	scanIntervalValue string,
) *releasetrackerworker.Scheduler {
	planReleaseNotifications := releasetrackerusecase.NewPlanReleaseNotifications(
		log,
		subscriptionRepo,
		notificationPublisher,
	)
	scanReleases := releasetrackerusecase.NewScanReleases(
		log,
		repositoryRepo,
		githubClient,
		planReleaseNotifications,
	)

	scanInterval, err := time.ParseDuration(scanIntervalValue)
	if err != nil {
		log.Error("failed to parse scanner interval", "val", scanIntervalValue, "err", err)
		scanInterval = time.Hour
	}

	return releasetrackerworker.NewScheduler(log, scanReleases, scanInterval)
}
