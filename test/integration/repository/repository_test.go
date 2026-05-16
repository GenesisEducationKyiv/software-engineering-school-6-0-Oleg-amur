//go:build integration

package repository_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/repository/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/test/integration/testkit"
)

var suite *testkit.Suite

func TestMain(m *testing.M) {
	os.Exit(testkit.Run(m, func(s *testkit.Suite) {
		suite = s
	}))
}

func TestRepositoryRepository_CreateAndGetByName(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	repo := postgresql.NewRepositoryRepository(suite.DB)

	created, err := repo.Create(ctx, "owner/repo", "v1.0.0")
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("want created repository to have an id")
	}
	if created.Name != "owner/repo" {
		t.Fatalf("got repository name %q, want %q", created.Name, "owner/repo")
	}
	if created.LastSeenTag != "v1.0.0" {
		t.Fatalf("got last seen tag %q, want %q", created.LastSeenTag, "v1.0.0")
	}

	got, err := repo.GetByName(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("get repository by name: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("got repository id %d, want %d", got.ID, created.ID)
	}
}

func TestRepositoryRepository_UpdateTag(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	repo := postgresql.NewRepositoryRepository(suite.DB)

	created, err := repo.Create(ctx, "owner/repo", "v1.0.0")
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}

	if err := repo.UpdateTag(ctx, created.ID, "v2.0.0"); err != nil {
		t.Fatalf("update repository tag: %v", err)
	}
	updated, err := repo.GetByName(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("get updated repository by name: %v", err)
	}
	if updated.LastSeenTag != "v2.0.0" {
		t.Fatalf("got updated tag %q, want %q", updated.LastSeenTag, "v2.0.0")
	}
}

func TestRepositoryRepository_GetAll(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	repo := postgresql.NewRepositoryRepository(suite.DB)

	if _, err := repo.Create(ctx, "owner/repo", "v1.0.0"); err != nil {
		t.Fatalf("create repository: %v", err)
	}

	all, err := repo.GetAll(ctx)
	if err != nil {
		t.Fatalf("get all repositories: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("got %d repositories, want 1", len(all))
	}
	if all[0].Name != "owner/repo" {
		t.Fatalf("got repository name %q, want %q", all[0].Name, "owner/repo")
	}
}

func TestRepositoryRepository_GetByNameReturnsNotFound(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	repo := postgresql.NewRepositoryRepository(suite.DB)

	_, err := repo.GetByName(ctx, "missing/repo")

	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, apperr.ErrNotFound)
	}
}
