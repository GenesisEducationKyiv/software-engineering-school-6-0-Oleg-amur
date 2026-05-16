//go:build integration

package testkit

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"testing"
	"time"

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

	db, log, err := startDatabase(ctx, container)
	if err != nil {
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
