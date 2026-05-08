package api

import (
	"log/slog"
	"net/http"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(log *slog.Logger, svc *service.SubscriptionService) http.Handler {
	mux := http.NewServeMux()
	h := NewHandler(log, svc)

	mux.HandleFunc("/api/v1/subscribe", h.Subscribe)
	mux.HandleFunc("/api/v1/confirm/", h.Confirm)
	mux.HandleFunc("/api/v1/unsubscribe/", h.Unsubscribe)
	mux.HandleFunc("/api/v1/subscriptions", h.GetSubscriptions)

	mux.Handle("/metrics", promhttp.Handler())

	return mux
}
