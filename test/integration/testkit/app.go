//go:build integration

package testkit

import (
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"testing"

	grpcapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/grpc"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/grpc/pb"
	httpapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/http"
	githubclient "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/client/github"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/contracts/events"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/eventbus/inmemory"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/repository/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/service"
	"github.com/stretchr/testify/require"
)

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

	subscriberService := service.NewSubscriberService(cfg.Logger, subscriberRepo)
	repositoryService := service.NewRepositoryService(cfg.Logger, repositoryRepo, githubClient)
	subscriptionEvents := make(chan events.SubscriptionConfirmationRequested, 10)
	releaseEvents := make(chan events.ReleaseNotificationRequested, 10)
	eventPublisher := inmemory.NewNotificationPublisher(subscriptionEvents, releaseEvents)
	subscriptionService := service.NewSubscriptionService(
		cfg.Logger,
		subscriberService,
		repositoryService,
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
