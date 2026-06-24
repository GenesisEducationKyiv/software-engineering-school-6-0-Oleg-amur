package observability

import (
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	statusCode int
	written    bool
}

func (r *statusRecorder) WriteHeader(statusCode int) {
	if r.written {
		return
	}
	r.statusCode = statusCode
	r.written = true
	r.ResponseWriter.WriteHeader(statusCode)
}

func HTTPMiddleware(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		recorder := &statusRecorder{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(recorder, r)

		route := r.Pattern
		if route == "" {
			route = r.URL.Path
		}
		status := strconv.Itoa(recorder.statusCode)
		duration := time.Since(start)
		httpRequestsTotal.WithLabelValues(r.Method, route, status).Inc()
		httpRequestDuration.WithLabelValues(r.Method, route, status).Observe(duration.Seconds())
		if recorder.statusCode >= http.StatusBadRequest {
			httpRequestErrorsTotal.WithLabelValues(r.Method, route, status).Inc()
		}

		log.Info(
			"http request completed",
			"method", r.Method,
			"route", route,
			"path", r.URL.Path,
			"status", recorder.statusCode,
			"duration_ms", duration.Milliseconds(),
			"remote_addr", r.RemoteAddr,
			"user_agent", r.UserAgent(),
		)
	})
}
