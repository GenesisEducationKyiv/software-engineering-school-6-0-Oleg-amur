package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log/slog"
	"sort"
	"strings"

	"github.com/Oleg-amur/case-task-swe-school-6.0/migrations"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func InitDb(ctx context.Context, connectionString string, log *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("pgx", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("database connection established succesfully")
	return db, nil
}

func RunMigrations(ctx context.Context, db *sql.DB, log *slog.Logger) error {
	files, err := fs.ReadDir(migrations.Files, ".")
	if err != nil {
		return fmt.Errorf("failed to read migrations: %w", err)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].Name() < files[j].Name()
	})

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		log.Info("applying migration", "file", file.Name())

		content, err := fs.ReadFile(migrations.Files, file.Name())

		if err != nil {
			return fmt.Errorf("failed to read a migration file: %w", err)
		}

		_, err = db.ExecContext(ctx, string(content))
		if err != nil {
			return fmt.Errorf("failed to apply a migration: %w", err)
		}
	}

	return nil
}
