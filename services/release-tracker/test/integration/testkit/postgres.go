//go:build integration

package testkit

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/database"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	postgresImage    = "postgres:18.3-alpine"
	testDatabaseName = "release_tracker_test"
	testDatabaseUser = "test"
	testDatabasePass = "test"
)

type Postgres struct {
	DB        *sql.DB
	Container *postgres.PostgresContainer
}

func NewPostgres(ctx context.Context, t testing.TB) *Postgres {
	t.Helper()

	container, err := postgres.Run(
		ctx,
		postgresImage,
		postgres.WithDatabase(testDatabaseName),
		postgres.WithUsername(testDatabaseUser),
		postgres.WithPassword(testDatabasePass),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "start postgres container")

	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		t.Fatalf("get postgres connection string: %v", err)
	}
	db, err := database.Open(ctx, connectionString)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		t.Fatalf("open postgres database: %v", err)
	}
	if err := database.Migrate(ctx, db); err != nil {
		_ = db.Close()
		_ = testcontainers.TerminateContainer(container)
		t.Fatalf("migrate postgres database: %v", err)
	}

	pg := &Postgres{DB: db, Container: container}
	t.Cleanup(func() {
		require.NoError(t, pg.Close(), "close postgres")
	})
	return pg
}

func (p *Postgres) Reset(t testing.TB) {
	t.Helper()
	_, err := p.DB.ExecContext(
		context.Background(),
		`TRUNCATE repositories RESTART IDENTITY CASCADE`,
	)
	require.NoError(t, err, "reset integration test database")
}

func (p *Postgres) Close() error {
	var errs []error
	if p.DB != nil {
		errs = append(errs, p.DB.Close())
	}
	if p.Container != nil {
		errs = append(errs, testcontainers.TerminateContainer(p.Container))
	}
	return errors.Join(errs...)
}
