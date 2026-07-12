package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

const healthCheckTimeout = time.Second

type HealthChecker interface {
	PingContext(context.Context) error
}

type healthResponse struct {
	Status string                `json:"status"`
	Checks map[string]checkState `json:"checks"`
}

type checkState struct {
	Status     string `json:"status"`
	DurationMS int64  `json:"duration_ms"`
	Error      string `json:"error,omitempty"`
}

func NewHealthHandler(log *slog.Logger, db HealthChecker) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		start := time.Now()
		ctx, cancel := context.WithTimeout(r.Context(), healthCheckTimeout)
		defer cancel()

		dbState := checkState{Status: "ok"}
		statusCode := http.StatusOK
		if err := db.PingContext(ctx); err != nil {
			dbState.Status = "down"
			dbState.Error = err.Error()
			statusCode = http.StatusServiceUnavailable
			log.Error("database health check failed", "err", err)
		}
		dbState.DurationMS = time.Since(start).Milliseconds()

		status := "ok"
		if statusCode != http.StatusOK {
			status = "degraded"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(healthResponse{
			Status: status,
			Checks: map[string]checkState{
				"database": dbState,
			},
		}); err != nil {
			log.Error("failed to encode health response", "err", err)
		}
	})
}
