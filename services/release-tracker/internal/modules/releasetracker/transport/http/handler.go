package releasetrackerhttp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/apperr"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/domain"
)

type RepositoryUsecases interface {
	EnsureTracked(context.Context, string) (*domain.Repository, error)
	GetRepository(context.Context, int64) (*domain.Repository, error)
}

type Handler struct {
	log      *slog.Logger
	usecases RepositoryUsecases
}

func NewHandler(log *slog.Logger, usecases RepositoryUsecases) *Handler {
	return &Handler{log: log, usecases: usecases}
}

func (h *Handler) EnsureTracked(w http.ResponseWriter, r *http.Request) {
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

	tracked, err := h.usecases.EnsureTracked(r.Context(), request.Repository)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeRepository(w, tracked)
}

func (h *Handler) GetRepository(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.URL.Query().Get("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusBadRequest, "positive repository id is required")
		return
	}
	tracked, err := h.usecases.GetRepository(r.Context(), id)
	if err != nil {
		h.handleError(w, err)
		return
	}
	writeRepository(w, tracked)
}

func (h *Handler) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, apperr.ErrRepositoryNotFound):
		writeError(w, http.StatusNotFound, "repository not found")
	case errors.Is(err, apperr.ErrRateLimitExceeded):
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
		ID          int64  `json:"id"`
		Name        string `json:"name"`
		LastSeenTag string `json:"last_seen_tag"`
	}{ID: tracked.ID, Name: tracked.Name, LastSeenTag: tracked.LastSeenTag})
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(struct {
		Message string `json:"message"`
	}{Message: message})
}
