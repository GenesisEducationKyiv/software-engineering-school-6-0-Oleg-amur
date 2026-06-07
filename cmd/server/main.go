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

	grpcapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/grpc"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/grpc/pb"
	httpapi "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/http"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/client/github"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/database"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/eventbus/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/observability"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/repository/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/scanner"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/service"
	"google.golang.org/grpc"
)

const (
	configPath             = "configs/config.yaml"
	errorChannelBufferSize = 2
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "release-notifier")
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
	githubClient, err := setupGithubClient(cfg.GithubClient, log)
	if err != nil {
		return err
	}

	subscriberRepo := postgresql.NewSubscriberRepository(db)
	repositoryRepo := postgresql.NewRepositoryRepository(db)
	subscriptionRepo := postgresql.NewSubscriptionRepository(db)

	notificationPublisher, err := rabbitmq.NewNotificationPublisher(rabbitMQConfig(cfg.EventBus))
	if err != nil {
		return err
	}
	defer func() {
		if err := notificationPublisher.Close(); err != nil {
			log.Error("unable to close notification publisher", "error", err)
		}
	}()

	subscriberService := service.NewSubscriberService(
		log,
		subscriberRepo,
	)

	repositoryService := service.NewRepositoryService(
		log,
		repositoryRepo,
		githubClient,
	)

	subscriptionSvc := service.NewSubscriptionService(
		log,
		subscriberService,
		repositoryService,
		subscriptionRepo,
		notificationPublisher,
	)

	releaseNotificationPlanner := service.NewReleaseNotificationPlanner(log, subscriptionRepo, notificationPublisher)
	releaseScanner := scanner.NewScanner(log, repositoryRepo, githubClient, releaseNotificationPlanner)

	scanInterval, err := time.ParseDuration(cfg.Scanner.Interval)
	if err != nil {
		log.Error("failed to parse scanner interval", "val", cfg.Scanner.Interval, "err", err)
		scanInterval = time.Hour
	}
	scheduler := scanner.NewScheduler(log, releaseScanner, scanInterval)
	go scheduler.Start(ctx)

	log.Debug("setting up transport layers")
	healthHandler := httpapi.NewHealthHandler(log, db)
	router := httpapi.NewRouter(log, subscriptionSvc, healthHandler)
	httpServer := setupHttpServer(cfg.Server, router)

	grpcHandler := grpcapi.NewGrpcHandler(log, subscriptionSvc)
	grpcServer, grpcLis, err := setupGrpcServer(ctx, cfg.Server, grpcHandler, log)
	if err != nil {
		return err
	}

	errCh := make(chan error, errorChannelBufferSize)

	go func() {
		log.Info("HTTP server starting", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	go func() {
		log.Info("gRPC server starting", "addr", ":"+cfg.Server.GrpcPort)
		if err := grpcServer.Serve(grpcLis); err != nil {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down signal received")
	case err := <-errCh:
		log.Error("server error", "error", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Info("shutting down servers...")
	grpcServer.GracefulStop()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return err
	}

	log.Info("graceful shutdown complete")

	return nil
}

func rabbitMQConfig(cfg config.EventBus) rabbitmq.Config {
	return rabbitmq.Config{
		URL:      cfg.URL,
		Exchange: cfg.NotificationExchange,
		Queue:    cfg.NotificationQueue,
		DLQ:      cfg.NotificationDLQ,
	}
}

func setupGithubClient(cfg config.GithubClient, log *slog.Logger) (*github.Client, error) {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse github client timeout: %w", err)
	}

	httpClient := &http.Client{Timeout: timeout}

	return github.NewClient(httpClient, cfg.Url, cfg.ApiToken, log), nil
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
	handler *grpcapi.GrpcHandler,
	log *slog.Logger,
) (*grpc.Server, net.Listener, error) {
	grpcAddr := ":" + cfg.GrpcPort

	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", grpcAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen for gRPC: %w", err)
	}

	srv := grpc.NewServer(grpc.UnaryInterceptor(observability.UnaryServerInterceptor(log)))
	pb.RegisterReleaseNotifierServer(srv, handler)

	return srv, lis, nil
}
