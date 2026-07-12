package api

import (
	"log/slog"
	"net/http"

	releasetrackerhttp "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/transport/http"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/observability"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(
	log *slog.Logger,
	usecases releasetrackerhttp.RepositoryUsecases,
	healthHandler http.Handler,
) http.Handler {
	handler := releasetrackerhttp.NewHandler(log, usecases)
	mux := http.NewServeMux()
	mux.HandleFunc("POST /internal/v1/repositories/ensure", handler.EnsureTracked)
	mux.HandleFunc("GET /internal/v1/repositories", handler.GetRepository)
	mux.Handle("GET /health", healthHandler)
	mux.Handle("GET /metrics", promhttp.Handler())
	return observability.HTTPMiddleware(log, mux)
}
