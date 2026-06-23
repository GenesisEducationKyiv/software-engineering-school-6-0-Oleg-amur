//go:build integration

package testkit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/eventbus/inmemory"
	postgresqladapter "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/adapters/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/grpc/pb"
	httpapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/api/http"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-notifier/internal/apperr"
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
	getOrCreateSubscriber := subscriptionusecase.NewGetOrCreateSubscriber(cfg.Logger, subscriberStore)
	repositoryTracker := &testRepositoryTracker{httpClient: gitHubHTTPClient, baseURL: cfg.GitHubURL}
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
			repositoryTracker,
			subscriptionStarter,
		),
		ConfirmSubscription:       subscriptionusecase.NewConfirmSubscription(subscriptionStore),
		UnsubscribeFromRepository: subscriptionusecase.NewUnsubscribeFromRepository(subscriptionStore),
		ListSubscriptions:         subscriptionusecase.NewListSubscriptions(subscriptionStore, repositoryTracker),
		ListActiveByRepository:    subscriptionusecase.NewListActiveSubscriptionsByRepository(subscriptionStore),
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
	subID int,
	repoName, email string,
	token string,
) error {
	if err := s.starter.StartSubscriptionConfirmation(ctx, subID, repoName, email, token); err != nil {
		return err
	}
	s.relay.Execute(ctx)
	return nil
}

type testRepositoryTracker struct {
	httpClient *http.Client
	baseURL    string
}

func (t *testRepositoryTracker) EnsureTracked(
	ctx context.Context,
	repoName string,
) (*subscriptionusecase.RepositoryView, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.baseURL+"/repos/"+repoName, nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusNotFound:
		return nil, apperr.ErrRepoNotFound
	case http.StatusTooManyRequests:
		return nil, apperr.ErrRateLimitExceeded
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub returned status %d", resp.StatusCode)
	}
	return t.GetRepository(ctx, repoName)
}

func (t *testRepositoryTracker) GetRepository(
	ctx context.Context,
	repoName string,
) (*subscriptionusecase.RepositoryView, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.baseURL+"/repos/"+repoName+"/releases/latest", nil)
	if err != nil {
		return nil, err
	}
	resp, err := t.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, apperr.ErrRepoNotFound
	}
	var response struct {
		Tag string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}
	return &subscriptionusecase.RepositoryView{Name: repoName, LastSeenTag: response.Tag}, nil
}
