package usecase

import (
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/domain"
)

func TestGetRepositoryReturnsTrackedRepository(t *testing.T) {
	want := &domain.Repository{ID: 7, Name: "owner/repo", LastSeenTag: "v1.0.0"}
	store := &repositoryStoreFake{repository: want}
	usecase := NewGetRepository(store)

	got, err := usecase.Execute(t.Context(), 7)
	if err != nil {
		t.Fatalf("get repository: %v", err)
	}
	if got != want {
		t.Fatalf("repository = %+v, want %+v", got, want)
	}
	if store.getID != 7 {
		t.Fatalf("repository ID = %d, want 7", store.getID)
	}
}
