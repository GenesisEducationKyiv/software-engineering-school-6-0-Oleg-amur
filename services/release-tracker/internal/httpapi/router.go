package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/domain"
	githubclient "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/github"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/repository"
)

type RepositoryService interface {
	EnsureTracked(context.Context, string) (*domain.Repository, error)
	GetRepository(context.Context, string) (*domain.Repository, error)
}

type Handler struct {
	log     *slog.Logger
	db      *sql.DB
	service RepositoryService
}

func NewRouter(log *slog.Logger, db *sql.DB, service RepositoryService) http.Handler {
	handler := &Handler{log: log, db: db, service: service}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /internal/v1/repositories/ensure", handler.ensureTracked)
	mux.HandleFunc("GET /internal/v1/repositories", handler.getRepository)
	mux.HandleFunc("GET /health", handler.health)
	return mux
}

func (h *Handler) ensureTracked(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Repository string `json:"repository"`
	}
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	if !validRepository(request.Repository) {
		writeError(w, http.StatusBadRequest, "repository must have owner/repo format")
		return
	}

	tracked, err := h.service.EnsureTracked(r.Context(), request.Repository)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeRepository(w, tracked)
}

func (h *Handler) getRepository(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("repository")
	if !validRepository(name) {
		writeError(w, http.StatusBadRequest, "repository must have owner/repo format")
		return
	}
	tracked, err := h.service.GetRepository(r.Context(), name)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeRepository(w, tracked)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if err := h.db.PingContext(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database unavailable")
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound), errors.Is(err, githubclient.ErrNotFound):
		writeError(w, http.StatusNotFound, "repository not found")
	case errors.Is(err, githubclient.ErrRateLimit):
		writeError(w, http.StatusTooManyRequests, "GitHub rate limit exceeded")
	default:
		h.log.Error("repository request failed", "err", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func validRepository(value string) bool {
	parts := strings.Split(value, "/")
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

func writeRepository(w http.ResponseWriter, tracked *domain.Repository) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Name        string `json:"name"`
		LastSeenTag string `json:"last_seen_tag"`
	}{Name: tracked.Name, LastSeenTag: tracked.LastSeenTag})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Message string `json:"message"`
	}{Message: message})
}
