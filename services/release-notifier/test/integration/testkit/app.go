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
	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/grpc/pb"
	httpapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/http"
	releasetrackerpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/persistence/postgresql"
	releasetrackerusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/releasetracker/usecase"
	subscriptionpostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/persistence/postgresql"
	subscriptiongrpc "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/transport/grpc"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/usecase"
	subscriptionworkflow "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/modules/subscriptions/workflow"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/contracts/events"
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

	queryable := postgresqladapter.NewContextQueryable(cfg.DB)
	transactionManager := postgresqladapter.NewTransactionManager(cfg.DB)
	subscriberStore := subscriptionpostgresql.NewSubscriberStore(queryable)
	repositoryStore := releasetrackerpostgresql.NewRepositoryStore(queryable)
	subscriptionStore := subscriptionpostgresql.NewSubscriptionStore(queryable)
	subscriptionSagaStore := subscriptionpostgresql.NewSubscriptionSagaStore(queryable)
	subscriptionOutboxStore := subscriptionpostgresql.NewOutboxStore(queryable)
	subscriptionSaga := subscriptionworkflow.NewSubscriptionConfirmationSaga(
		cfg.Logger,
		transactionManager,
		subscriptionStore,
		subscriptionSagaStore,
		subscriptionOutboxStore,
	)
	githubClient := githubclient.NewClient(gitHubHTTPClient, cfg.GitHubURL, "test-token", cfg.Logger)

	getOrCreateSubscriber := subscriptionusecase.NewGetOrCreateSubscriber(cfg.Logger, subscriberStore)
	ensureRepositoryTracked := releasetrackerusecase.NewEnsureRepositoryTracked(cfg.Logger, repositoryStore, githubClient)
	subscriptionEvents := make(chan events.SubscriptionConfirmationRequested, 10)
	releaseEvents := make(chan events.ReleaseNotificationRequested, 10)
	eventPublisher := inmemory.NewNotificationPublisher(subscriptionEvents, releaseEvents)
	subscriptionStarter := &testSubscriptionStarter{
		starter: subscriptionSaga,
		relay: subscriptionworkflow.NewPublishSubscriptionOutbox(
			cfg.Logger,
			subscriptionOutboxStore,
			eventPublisher,
		),
	}
	subscriptionUsecases := subscriptionusecase.SubscriptionUsecases{
		SubscribeToRepository: subscriptionusecase.NewSubscribeToRepository(
			cfg.Logger,
			getOrCreateSubscriber,
			ensureRepositoryTracked,
			subscriptionStarter,
		),
		ConfirmSubscription:       subscriptionusecase.NewConfirmSubscription(subscriptionStore),
		UnsubscribeFromRepository: subscriptionusecase.NewUnsubscribeFromRepository(subscriptionStore),
		ListSubscriptions:         subscriptionusecase.NewListSubscriptions(subscriptionStore),
	}

	return &App{
		DB:          cfg.DB,
		Logger:      cfg.Logger,
		HTTPHandler: httpapi.NewRouter(cfg.Logger, subscriptionUsecases, httpapi.NewHealthHandler(cfg.Logger, cfg.DB)),
		GRPCHandler: subscriptiongrpc.NewHandler(cfg.Logger, subscriptionUsecases),
		Events:      subscriptionEvents,
	}
}

type testSubscriptionStarter struct {
	starter *subscriptionworkflow.SubscriptionConfirmationSaga
	relay   *subscriptionworkflow.PublishSubscriptionOutbox
}

func (s *testSubscriptionStarter) StartSubscriptionConfirmation(
	ctx context.Context,
	subID, repoID int,
	email string,
	token string,
) error {
	if err := s.starter.StartSubscriptionConfirmation(ctx, subID, repoID, email, token); err != nil {
		return err
	}
	s.relay.Execute(ctx)
	return nil
}
