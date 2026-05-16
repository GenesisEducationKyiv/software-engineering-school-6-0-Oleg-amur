//go:build integration

package testkit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/database"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/model"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/repository/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/service"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	postgresImage    = "postgres:18.3-alpine"
	testDatabaseName = "release_notifier_test"
	testDatabaseUser = "test"
	testDatabasePass = "test"
)

type Suite struct {
	DB        *sql.DB
	Postgres  *postgres.PostgresContainer
	Logger    *slog.Logger
	terminate func() error
}

func Run(m *testing.M, setSuite func(*Suite)) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	suite, err := Start(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start integration test suite: %v\n", err)
		return 1
	}
	setSuite(suite)

	code := m.Run()
	if err := suite.Close(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to close integration test suite: %v\n", err)
		code = 1
	}

	return code
}

func Start(ctx context.Context) (*Suite, error) {
	container, err := postgres.Run(
		ctx,
		postgresImage,
		postgres.WithDatabase(testDatabaseName),
		postgres.WithUsername(testDatabaseUser),
		postgres.WithPassword(testDatabasePass),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		return nil, err
	}

	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, err
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	db, err := database.InitDb(ctx, connString, log)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		return nil, err
	}

	if err := database.RunMigrations(ctx, db, log); err != nil {
		_ = db.Close()
		_ = testcontainers.TerminateContainer(container)
		return nil, err
	}

	return &Suite{
		DB:       db,
		Postgres: container,
		Logger:   log,
		terminate: func() error {
			return testcontainers.TerminateContainer(container)
		},
	}, nil
}

func (s *Suite) Close() error {
	var errs []error
	if s.DB != nil {
		if err := s.DB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if s.terminate != nil {
		if err := s.terminate(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *Suite) ResetDatabase(t testing.TB) {
	t.Helper()

	_, err := s.DB.ExecContext(
		context.Background(),
		"TRUNCATE subscriptions, subscribers, repositories RESTART IDENTITY CASCADE",
	)
	if err != nil {
		t.Fatalf("reset integration test database: %v", err)
	}
}

func (s *Suite) NewSubscriptionService() (
	*service.SubscriptionService,
	*FakeGithubClient,
	chan model.SubscriptionEvent,
) {
	subscriberRepo := postgresql.NewSubscriberRepository(s.DB)
	repositoryRepo := postgresql.NewRepositoryRepository(s.DB)
	subscriptionRepo := postgresql.NewSubscriptionRepository(s.DB)

	githubClient := &FakeGithubClient{
		Exists: map[string]bool{
			"owner/repo": true,
		},
		Tags: map[string]string{
			"owner/repo": "v1.0.0",
		},
		CheckErr: map[string]error{},
		TagErr:   map[string]error{},
	}

	subscriberService := service.NewSubscriberService(s.Logger, subscriberRepo)
	repositoryService := service.NewRepositoryService(s.Logger, repositoryRepo, githubClient)
	subscriptionEvents := make(chan model.SubscriptionEvent, 10)
	subscriptionService := service.NewSubscriptionService(
		s.Logger,
		subscriberService,
		repositoryService,
		subscriptionRepo,
		subscriptionEvents,
	)

	return subscriptionService, githubClient, subscriptionEvents
}

type FakeGithubClient struct {
	Exists   map[string]bool
	Tags     map[string]string
	CheckErr map[string]error
	TagErr   map[string]error
}

func (f *FakeGithubClient) CheckIfRepoExists(ctx context.Context, repoAddr string) (bool, error) {
	if err := f.CheckErr[repoAddr]; err != nil {
		return false, err
	}
	return f.Exists[repoAddr], nil
}

func (f *FakeGithubClient) GetRepositoryLatestTag(ctx context.Context, repoAddr string) (string, error) {
	if err := f.TagErr[repoAddr]; err != nil {
		return "", err
	}
	return f.Tags[repoAddr], nil
}
