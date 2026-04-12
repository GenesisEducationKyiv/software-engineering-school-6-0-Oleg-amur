package api

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/api/http/dto"
	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/apperr"
	"github.com/Oleg-amur/case-task-swe-school-6.0/internal/service"
)

type Handler struct {
	log     *slog.Logger
	service *service.SubscriptionService
}

func NewHandler(log *slog.Logger, svc *service.SubscriptionService) *Handler {
	return &Handler{
		log:     log,
		service: svc,
	}
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req dto.SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	err := h.service.Subscribe(r.Context(), req)
	if err != nil {
		if errors.Is(err, apperr.ErrInvalidFormat) {
			h.sendError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, apperr.ErrRepoNotFound) {
			h.sendError(w, err.Error(), http.StatusNotFound)
			return
		}
		if errors.Is(err, apperr.ErrRateLimitExceeded) {
			h.sendError(w, err.Error(), http.StatusTooManyRequests)
			return
		}
		if errors.Is(err, apperr.ErrAlreadySubscribed) {
			h.sendError(w, err.Error(), http.StatusConflict)
			return
		}
		h.log.Error("subscription failed", "err", err)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	token := strings.TrimPrefix(r.URL.Path, "/api/confirm/")
	if token == "" {
		h.sendError(w, "Missing token", http.StatusBadRequest)
		return
	}

	err := h.service.Confirm(r.Context(), token)
	if err != nil {
		if errors.Is(err, apperr.ErrTokenNotFound) {
			h.sendError(w, "Token not found", http.StatusNotFound)
			return
		}
		h.log.Error("confirmation failed", "err", err)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) Unsubscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	token := strings.TrimPrefix(r.URL.Path, "/api/unsubscribe/")
	if token == "" {
		h.sendError(w, "Missing token", http.StatusBadRequest)
		return
	}

	err := h.service.Unsubscribe(r.Context(), token)
	if err != nil {
		h.log.Error("unsubscription failed", "err", err)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) GetSubscriptions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	email := r.URL.Query().Get("email")
	if email == "" {
		h.sendError(w, "Email parameter is required", http.StatusBadRequest)
		return
	}

	subs, err := h.service.GetSubscriptions(r.Context(), email)
	if err != nil {
		h.log.Error("get subscriptions failed", "err", err)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(subs)
}

func (h *Handler) sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(dto.ErrorResponse{Message: message})
}
