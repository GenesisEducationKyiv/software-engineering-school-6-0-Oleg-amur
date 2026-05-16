//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/repository/postgresql"
)

func TestSubscriberRepository_CreateAndGetByEmail(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	repo := postgresql.NewSubscriberRepository(suite.DB)

	created, err := repo.Create(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("create subscriber: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("want created subscriber to have an id")
	}
	if created.Email != "user@example.com" {
		t.Fatalf("got email %q, want %q", created.Email, "user@example.com")
	}

	got, err := repo.GetByEmail(ctx, "user@example.com")
	if err != nil {
		t.Fatalf("get subscriber by email: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("got subscriber id %d, want %d", got.ID, created.ID)
	}
}

func TestSubscriberRepository_GetByEmailReturnsNotFound(t *testing.T) {
	suite.ResetDatabase(t)
	ctx := context.Background()
	repo := postgresql.NewSubscriberRepository(suite.DB)

	_, err := repo.GetByEmail(ctx, "missing@example.com")

	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("got error %v, want %v", err, apperr.ErrNotFound)
	}
}
