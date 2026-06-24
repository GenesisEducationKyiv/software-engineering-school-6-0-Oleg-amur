package subscriptionhttp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/apperr"
	subscriptiondomain "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/domain"
	subscriptionusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/usecase"
)

type SubscriptionUsecases interface {
	Subscribe(context.Context, subscriptionusecase.SubscribeRequest) error
	Confirm(context.Context, string) error
	Unsubscribe(context.Context, string) error
	GetSubscriptions(context.Context, string) ([]subscriptionusecase.SubscriptionView, error)
	GetActiveSubscriptionsByRepository(context.Context, int64) ([]subscriptiondomain.RepositorySubscription, error)
}

func (h *Handler) GetActiveSubscriptionsByRepository(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	repositoryID, err := strconv.ParseInt(r.URL.Query().Get("repository_id"), 10, 64)
	if err != nil || repositoryID <= 0 {
		h.sendError(w, "Positive repository_id parameter is required", http.StatusBadRequest)
		return
	}

	subscriptions, err := h.usecases.GetActiveSubscriptionsByRepository(r.Context(), repositoryID)
	if err != nil {
		h.log.Error("get active subscriptions by repository failed", "repository_id", repositoryID, "err", err)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := ActiveSubscriptionsResponse{Subscriptions: make([]ActiveSubscription, 0, len(subscriptions))}
	for _, subscription := range subscriptions {
		response.Subscriptions = append(response.Subscriptions, ActiveSubscription{
			Email:            subscription.Email,
			UnsubscribeToken: subscription.UnsubscribeToken,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("failed to encode active subscriptions response", "err", err)
	}
}

type Handler struct {
	log      *slog.Logger
	usecases SubscriptionUsecases
}

func NewHandler(log *slog.Logger, usecases SubscriptionUsecases) *Handler {
	return &Handler{
		log:      log,
		usecases: usecases,
	}
}

func (h *Handler) Subscribe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SubscribeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.sendError(w, "Invalid JSON payload", http.StatusBadRequest)
		return
	}

	subReq := subscriptionusecase.SubscribeRequest{
		Email: req.Email,
		Repo:  req.Repo,
	}

	if err := subReq.Validate(); err != nil {
		h.sendError(w, err.Error(), http.StatusBadRequest)
		return
	}

	err := h.usecases.Subscribe(r.Context(), subReq)
	if err != nil {
		if errors.Is(err, apperr.ErrInvalidEmailFormat) ||
			errors.Is(err, apperr.ErrInvalidRepositoryFormat) {
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

	token := r.PathValue("token")
	if token == "" {
		h.sendError(w, "Missing token", http.StatusBadRequest)
		return
	}

	err := h.usecases.Confirm(r.Context(), token)
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

	token := r.PathValue("token")
	if token == "" {
		h.sendError(w, "Missing token", http.StatusBadRequest)
		return
	}

	err := h.usecases.Unsubscribe(r.Context(), token)
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

	subs, err := h.usecases.GetSubscriptions(r.Context(), email)
	if err != nil {
		h.log.Error("get subscriptions failed", "err", err)
		h.sendError(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	response := make([]Subscription, 0, len(subs))
	for _, s := range subs {
		response = append(response, Subscription{
			Email:       s.Email,
			Repo:        s.Repo,
			Confirmed:   s.Confirmed,
			LastSeenTag: s.LastSeenTag,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		h.log.Error("failed to encode response", "err", err)
		h.sendError(w, "failed to encode response", http.StatusInternalServerError)
	}
}

func (h *Handler) sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(ErrorResponse{Message: message}); err != nil {
		h.log.Error("failed to encode response", "err", err)
	}
}
