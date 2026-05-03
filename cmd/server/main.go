package main

import (
	"context"
	"database/sql"
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
	api "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/api/http"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/database"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/github"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/notifier"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/repository/postgresql"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/scanner"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/internal/service"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if err := runApp(log); err != nil {
		log.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func runApp(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadConfig("configs/config.yaml")
	if err != nil {
		return err
	}

	db, err := database.InitDb(ctx, cfg.Database.ConnectionString, log)
	if err != nil {
		return err
	}
	defer func(db *sql.DB) {
		err := db.Close()
		if err != nil {
			log.Error("unable to close database connection", "error", err)
		}
	}(db)

	if err := database.RunMigrations(ctx, db, log); err != nil {
		return err
	}

	githubClient, err := setupGithubClient(cfg.GithubClient)
	if err != nil {
		return err
	}

	subRepo := postgresql.NewSubscriberRepository(db)
	repoRepo := postgresql.NewRepositoryRepository(db)
	subscriptionRepo := postgresql.NewSubscriptionRepository(db)

	n := notifier.NewEmailNotifier(cfg.Notifier)
	subscriptionSvc := service.NewSubscriptionService(
		log,
		subRepo,
		repoRepo,
		subscriptionRepo,
		n,
		githubClient,
	)

	s := setupScanner(log, cfg.Scanner, repoRepo, subscriptionRepo, githubClient, n)
	go s.Start(ctx)

	h := api.NewHandler(log, subscriptionSvc)
	mux := setupMux(h)
	httpServer := setupHttpServer(cfg.Server, mux)

	grpcH := grpcapi.NewGrpcHandler(log, subscriptionSvc)
	grpcServer, grpcLis, err := setupGrpcServer(ctx, cfg.Server, grpcH)
	if err != nil {
		return err
	}

	errCh := make(chan error, 2)

	go func() {
		log.Info("HTTP server started", "addr", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	go func() {
		log.Info("gRPC server started", "addr", ":"+cfg.Server.GrpcPort)
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

func setupGithubClient(cfg config.GithubClient) (*github.Client, error) {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse github client timeout: %w", err)
	}
	return github.NewClient(cfg.Url, cfg.ApiToken, timeout), nil
}

func setupScanner(
	log *slog.Logger,
	cfg config.Scanner,
	repoRepo *postgresql.RepositoryRepository,
	subRepo *postgresql.SubscriptionRepository,
	ghClient *github.Client,
	n service.Notifier,
) *scanner.Scanner {
	scanInterval, err := time.ParseDuration(cfg.Interval)
	if err != nil {
		log.Error("failed to parse scanner interval", "val", cfg.Interval, "err", err)
		scanInterval = time.Hour
	}
	return scanner.NewScanner(log, repoRepo, subRepo, ghClient, n, scanInterval)
}

func setupHttpServer(cfg config.Server, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: handler,
	}
}

func setupGrpcServer(
	ctx context.Context,
	cfg config.Server,
	handler *grpcapi.GrpcHandler,
) (*grpc.Server, net.Listener, error) {
	grpcAddr := ":" + cfg.GrpcPort

	lc := net.ListenConfig{}
	lis, err := lc.Listen(ctx, "tcp", grpcAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to listen for gRPC: %w", err)
	}

	srv := grpc.NewServer()
	pb.RegisterReleaseNotifierServer(srv, handler)

	return srv, lis, nil
}

func setupMux(h *api.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/subscribe", h.Subscribe)
	mux.HandleFunc("/api/confirm/", h.Confirm)
	mux.HandleFunc("/api/unsubscribe/", h.Unsubscribe)
	mux.HandleFunc("/api/subscriptions", h.GetSubscriptions)
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}
