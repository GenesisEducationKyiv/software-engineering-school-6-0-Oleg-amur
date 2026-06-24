//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/apperr"
	repositorypostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/persistence/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/test/integration/testkit"
)

func TestRepositoryStoreLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pg := testkit.NewPostgres(ctx, t)
	pg.Reset(t)
	store := repositorypostgresql.NewRepositoryStore(pg.DB)

	created, err := store.Create(ctx, "owner/repo", "v1")
	if err != nil {
		t.Fatalf("create repository: %v", err)
	}
	if created.ID == 0 || created.Name != "owner/repo" || created.LastSeenTag != "v1" {
		t.Fatalf("unexpected created repository: %+v", created)
	}

	if err := store.UpdateTag(ctx, created.ID, "v2"); err != nil {
		t.Fatalf("update repository tag: %v", err)
	}
	got, err := store.GetByName(ctx, "owner/repo")
	if err != nil {
		t.Fatalf("get repository: %v", err)
	}
	if got.LastSeenTag != "v2" {
		t.Fatalf("last seen tag = %q, want v2", got.LastSeenTag)
	}

	repositories, err := store.GetAll(ctx)
	if err != nil {
		t.Fatalf("list repositories: %v", err)
	}
	if len(repositories) != 1 {
		t.Fatalf("repository count = %d, want 1", len(repositories))
	}

	_, err = store.GetByName(ctx, "missing/repo")
	if !errors.Is(err, apperr.ErrRepositoryNotFound) {
		t.Fatalf("missing repository error = %v", err)
	}
}
