package observability

import "github.com/prometheus/client_golang/prometheus"

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "subscription_service",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests by method, route, and status.",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "subscription_service",
			Subsystem: "http",
			Name:      "request_errors_total",
			Help:      "Total number of HTTP requests that ended with a 5xx status.",
		},
		[]string{"method", "route", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "subscription_service",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds by method, route, and status.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "route", "status"},
	)

	grpcRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "subscription_service",
			Subsystem: "grpc",
			Name:      "requests_total",
			Help:      "Total number of gRPC requests by method and status code.",
		},
		[]string{"method", "status_code"},
	)

	grpcRequestErrorsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "subscription_service",
			Subsystem: "grpc",
			Name:      "request_errors_total",
			Help:      "Total number of gRPC requests that ended with a server error status code.",
		},
		[]string{"method", "status_code"},
	)

	grpcRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "subscription_service",
			Subsystem: "grpc",
			Name:      "request_duration_seconds",
			Help:      "gRPC request duration in seconds by method and status code.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "status_code"},
	)
)

func init() {
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestErrorsTotal,
		httpRequestDuration,
		grpcRequestsTotal,
		grpcRequestErrorsTotal,
		grpcRequestDuration,
	)
}
