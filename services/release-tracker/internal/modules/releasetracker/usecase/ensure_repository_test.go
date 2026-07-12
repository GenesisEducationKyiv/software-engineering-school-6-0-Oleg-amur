package usecase

import (
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/apperr"
)

func TestEnsureRepositoryCreatesMissingRepository(t *testing.T) {
	store := &repositoryStoreFake{getErr: apperr.ErrRepositoryNotFound}
	github := &githubClientFake{exists: true, latestTag: "v1.0.0"}
	usecase := NewEnsureRepository(store, github)

	tracked, err := usecase.Execute(t.Context(), "owner/repo")
	if err != nil {
		t.Fatalf("ensure repository: %v", err)
	}
	if tracked.Name != "owner/repo" || tracked.LastSeenTag != "v1.0.0" {
		t.Fatalf("unexpected repository: %+v", tracked)
	}
	if store.created != "owner/repo" {
		t.Fatalf("created repository %q", store.created)
	}
}
