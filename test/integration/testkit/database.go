//go:build integration

package testkit

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/database"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

func startDatabase(ctx context.Context, container *postgres.PostgresContainer) (*sql.DB, *slog.Logger, error) {
	connString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, nil, err
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	db, err := database.InitDb(ctx, connString, log)
	if err != nil {
		return nil, nil, err
	}

	if err := database.RunMigrations(ctx, db, log); err != nil {
		_ = db.Close()
		return nil, nil, err
	}

	return db, log, nil
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
