package api

import (
	"log/slog"
	"net/http"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/observability"
	webstatic "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/static"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func NewRouter(log *slog.Logger, svc SubscriptionService, healthHandler http.Handler) http.Handler {
	mux := http.NewServeMux()
	h := NewHandler(log, svc)

	mux.HandleFunc("/", serveIndex)
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(webstatic.Files))))

	mux.HandleFunc("/api/v1/subscribe", h.Subscribe)
	mux.HandleFunc("/api/v1/confirm/{token}", h.Confirm)
	mux.HandleFunc("/api/v1/unsubscribe/{token}", h.Unsubscribe)
	mux.HandleFunc("/api/v1/subscriptions", h.GetSubscriptions)

	mux.Handle("/health", healthHandler)
	mux.Handle("/metrics", promhttp.Handler())

	return observability.HTTPMiddleware(log, mux)
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	http.ServeFileFS(w, r, webstatic.Files, "index.html")
}
