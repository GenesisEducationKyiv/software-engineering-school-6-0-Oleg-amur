//go:build integration

package testkit

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"testing"

	httpapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/api/http"
	repositorypostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/persistence/postgresql"
	releasetrackerusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/usecase"
)

type App struct {
	HTTPHandler http.Handler
}

func NewApp(t testing.TB, pg *Postgres, github *GitHub) *App {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	repositories := repositorypostgresql.NewRepositoryStore(pg.DB)
	usecases := releasetrackerusecase.Usecases{
		EnsureRepository: releasetrackerusecase.NewEnsureRepository(repositories, github),
		RepositoryQuery:  releasetrackerusecase.NewGetRepository(repositories),
	}
	return &App{
		HTTPHandler: httpapi.NewRouter(log, usecases, httpapi.NewHealthHandler(log, pg.DB)),
	}
}

type GitHub struct {
	Exists bool
	Tag    string
	Err    error
}

func (g *GitHub) RepositoryExists(context.Context, string) (bool, error) {
	return g.Exists, g.Err
}

func (g *GitHub) LatestTag(context.Context, string) (string, error) {
	return g.Tag, g.Err
}
