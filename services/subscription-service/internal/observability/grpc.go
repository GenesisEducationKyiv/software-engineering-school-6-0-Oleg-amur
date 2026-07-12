package observability

import (
	"context"
	"log/slog"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func UnaryServerInterceptor(log *slog.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		start := time.Now()

		resp, err := handler(ctx, req)

		duration := time.Since(start)
		code := status.Code(err).String()

		grpcRequestsTotal.WithLabelValues(info.FullMethod, code).Inc()
		grpcRequestDuration.WithLabelValues(info.FullMethod, code).Observe(duration.Seconds())
		if isServerErrorCode(status.Code(err)) {
			grpcRequestErrorsTotal.WithLabelValues(info.FullMethod, code).Inc()
		}

		log.Info(
			"grpc request completed",
			"method", info.FullMethod,
			"status_code", code,
			"duration_ms", duration.Milliseconds(),
		)

		return resp, err
	}
}

func isServerErrorCode(code codes.Code) bool {
	switch code {
	case codes.Unknown, codes.DeadlineExceeded, codes.Internal, codes.Unavailable, codes.DataLoss:
		return true
	default:
		return false
	}
}
