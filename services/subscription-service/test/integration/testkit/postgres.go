//go:build integration

package testkit

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/database"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	postgresImage    = "postgres:18.3-alpine"
	testDatabaseName = "subscription_service_test"
	testDatabaseUser = "test"
	testDatabasePass = "test"
)

type Postgres struct {
	DB        *sql.DB
	Container *postgres.PostgresContainer
}

func NewPostgres(ctx context.Context, t testing.TB) *Postgres {
	t.Helper()

	log := slog.New(slog.NewTextHandler(io.Discard, nil))

	container, err := postgres.Run(
		ctx,
		postgresImage,
		postgres.WithDatabase(testDatabaseName),
		postgres.WithUsername(testDatabaseUser),
		postgres.WithPassword(testDatabasePass),
		postgres.BasicWaitStrategies(),
	)
	require.NoError(t, err, "start postgres container")

	db, err := openPostgresDatabase(ctx, container, log)
	if err != nil {
		_ = testcontainers.TerminateContainer(container)
		t.Fatalf("start postgres database: %v", err)
	}

	pg := &Postgres{
		DB:        db,
		Container: container,
	}
	t.Cleanup(func() {
		require.NoError(t, pg.Close(), "close postgres")
	})

	return pg
}

func openPostgresDatabase(
	ctx context.Context,
	container *postgres.PostgresContainer,
	log *slog.Logger,
) (*sql.DB, error) {
	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, err
	}

	db, err := database.InitDb(ctx, connString, log)
	if err != nil {
		return nil, err
	}

	if err := database.RunMigrations(ctx, db, log); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func (p *Postgres) Reset(t testing.TB) {
	t.Helper()

	_, err := p.DB.ExecContext(
		context.Background(),
		`TRUNCATE
			outbox_messages,
			subscription_sagas,
			subscriptions,
			subscribers
		RESTART IDENTITY CASCADE`,
	)
	if err != nil {
		t.Fatalf("reset integration test database: %v", err)
	}
}

func (p *Postgres) Close() error {
	var errs []error
	if p.DB != nil {
		if err := p.DB.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if p.Container != nil {
		if err := testcontainers.TerminateContainer(p.Container); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}
