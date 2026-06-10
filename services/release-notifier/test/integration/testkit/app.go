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
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/postgresql"
	grpcapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/grpc"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/grpc/pb"
	httpapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/http"
	releasewatchservices "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasewatch/services"
	subscriptionmodels "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/models"
	subscriptionservices "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/services"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
	"github.com/stretchr/testify/require"
)

type repositoryTracker struct {
	service *releasewatchservices.RepositoryService
}

func (t repositoryTracker) EnsureTracked(
	ctx context.Context,
	repoName string,
) (*subscriptionmodels.TrackedRepositoryRef, error) {
	repo, err := t.service.EnsureTracked(ctx, repoName)
	if err != nil {
		return nil, err
	}

	return &subscriptionmodels.TrackedRepositoryRef{
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

	subscriberRepo := postgresql.NewSubscriberRepository(cfg.DB)
	repositoryRepo := postgresql.NewRepositoryRepository(cfg.DB)
	subscriptionRepo := postgresql.NewSubscriptionRepository(cfg.DB)
	githubClient := githubclient.NewClient(gitHubHTTPClient, cfg.GitHubURL, "test-token", cfg.Logger)

	subscriberService := subscriptionservices.NewSubscriberService(cfg.Logger, subscriberRepo)
	releasewatchService := releasewatchservices.NewRepositoryService(cfg.Logger, repositoryRepo, githubClient)
	subscriptionEvents := make(chan events.SubscriptionConfirmationRequested, 10)
	releaseEvents := make(chan events.ReleaseNotificationRequested, 10)
	eventPublisher := inmemory.NewNotificationPublisher(subscriptionEvents, releaseEvents)
	subscriptionService := subscriptionservices.NewSubscriptionService(
		cfg.Logger,
		subscriberService,
		repositoryTracker{service: releasewatchService},
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
