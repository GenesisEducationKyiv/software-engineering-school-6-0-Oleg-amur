//go:build integration

package testkit

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/eventbus/inmemory"
	githubclient "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/github"
	grpcapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/grpc"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/grpc/pb"
	httpapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/http"
	releasetrackerpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/persistence/postgresql"
	releasetrackerusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/usecase"
	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/domain"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/usecase"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/stretchr/testify/require"
)

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

type AppConfig struct {
	DB        *sql.DB
	Logger    *slog.Logger
	GitHubURL string
}

type App struct {
	DB          *sql.DB
	Logger      *slog.Logger
	HTTPHandler http.Handler
	GRPCHandler pb.ReleaseNotifierServer
	Events      chan events.SubscriptionConfirmationRequested
}

func NewApp(t testing.TB, cfg AppConfig) *App {
	t.Helper()

	require.NotNil(t, cfg.DB, "test app database is required")
	require.NotEmpty(t, cfg.GitHubURL, "test app GitHub URL is required")
	if cfg.Logger == nil {
		cfg.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}

	gitHubHTTPClient := &http.Client{}

	subscriberRepo := subscriptionpostgresql.NewSubscriberRepository(cfg.DB)
	repositoryRepo := releasetrackerpostgresql.NewRepositoryStore(cfg.DB)
	subscriptionRepo := subscriptionpostgresql.NewSubscriptionRepository(cfg.DB)
	githubClient := githubclient.NewClient(gitHubHTTPClient, cfg.GitHubURL, "test-token", cfg.Logger)

	subscriberService := subscriptionusecase.NewSubscriberService(cfg.Logger, subscriberRepo)
	releaseTrackerService := releasetrackerusecase.NewRepositoryService(cfg.Logger, repositoryRepo, githubClient)
	subscriptionEvents := make(chan events.SubscriptionConfirmationRequested, 10)
	releaseEvents := make(chan events.ReleaseNotificationRequested, 10)
	eventPublisher := inmemory.NewNotificationPublisher(subscriptionEvents, releaseEvents)
	subscriptionService := subscriptionusecase.NewSubscriptionService(
		cfg.Logger,
		subscriberService,
		repositoryTracker{service: releaseTrackerService},
		subscriptionRepo,
		eventPublisher,
	)

	return &App{
		DB:          cfg.DB,
		Logger:      cfg.Logger,
		HTTPHandler: httpapi.NewRouter(cfg.Logger, subscriptionService, httpapi.NewHealthHandler(cfg.Logger, cfg.DB)),
		GRPCHandler: grpcapi.NewGrpcHandler(cfg.Logger, subscriptionService),
		Events:      subscriptionEvents,
	}
}
