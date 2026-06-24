package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/api/grpc/pb"
	httpapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/api/http"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/database"
	subscriptiongrpc "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/modules/subscriptions/transport/grpc"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/subscription-service/internal/observability"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
)

const (
	configPath = "configs/config.yaml"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "subscription-service")
	slog.SetDefault(log)

	if err := runApp(log); err != nil {
		log.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func runApp(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Debug("loading configuration", "config_path", configPath)

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config from %s: %w", configPath, err)
	}

	log.Debug("connecting to database")
	db, err := database.InitDb(ctx, cfg.Database.ConnectionString, log)
	if err != nil {
		return err
	}
	if err := observability.RegisterDatabaseMetrics(db, log); err != nil {
		log.Error("failed to register database metrics; continuing without database metrics", "error", err)
	}
	defer func(db *sql.DB) {
		log.Debug("closing database connection")
		err := db.Close()
		if err != nil {
			log.Error("unable to close database connection", "error", err)
		}
	}(db)

	log.Debug("running database migrations")
	if err := database.RunMigrations(ctx, db, log); err != nil {
		return err
	}

	log.Debug("initializing dependencies")
	modules, err := setupModules(log, db, cfg)
	if err != nil {
		return err
	}
	defer func() {
		if err := modules.notificationPublisher.Close(); err != nil {
			log.Error("unable to close notification publisher", "error", err)
		}
		if err := modules.subscriptionSagaConsumer.Close(); err != nil {
			log.Error("unable to close subscription saga consumer", "error", err)
		}
	}()

	go modules.subscriptionOutboxRelay.Start(ctx)

	log.Debug("setting up transport layers")
	healthHandler := httpapi.NewHealthHandler(log, db)
	router := httpapi.NewRouter(log, modules.subscriptionUsecases, healthHandler)
	httpServer := setupHttpServer(cfg.Server, router)

	grpcHandler := subscriptiongrpc.NewHandler(log, modules.subscriptionUsecases)
	grpcServer, grpcLis, err := setupGrpcServer(ctx, cfg.Server, grpcHandler, log)
	if err != nil {
		return err
	}

	group, groupCtx := errgroup.WithContext(ctx)

	group.Go(func() error {
		log.Info("HTTP server starting", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP server failed: %w", err)
		}
		return nil
	})

	group.Go(func() error {
		log.Info("gRPC server starting", "addr", ":"+cfg.Server.GrpcPort)
		if err := grpcServer.Serve(grpcLis); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			return fmt.Errorf("gRPC server failed: %w", err)
		}
		return nil
	})

	group.Go(func() error {
		log.Info("subscription saga consumer starting", "queue", cfg.EventBus.SubscriptionSagaQueue)
		return modules.subscriptionSagaConsumer.Subscribe(groupCtx, modules.subscriptionSaga)
	})

	group.Go(func() error {
		<-groupCtx.Done()
		if errors.Is(ctx.Err(), context.Canceled) {
			log.Info("shutting down signal received")
		}

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		log.Info("shutting down servers...")
		grpcServer.GracefulStop()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("shutdown HTTP server: %w", err)
		}

		return nil
	})

	if err := group.Wait(); err != nil {
		return err
	}

	log.Info("graceful shutdown complete")

	return nil
}

func setupHttpServer(cfg config.Server, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}
}

func setupGrpcServer(
	ctx context.Context,
	cfg config.Server,
	handler *subscriptiongrpc.Handler,
	log *slog.Logger,
) (*grpc.Server, net.Listener, error) {
	grpcAddr := ":" + cfg.GrpcPort

	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", grpcAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen for gRPC: %w", err)
	}

	srv := grpc.NewServer(grpc.UnaryInterceptor(observability.UnaryServerInterceptor(log)))
	pb.RegisterSubscriptionServiceServer(srv, handler)

	return srv, lis, nil
}
