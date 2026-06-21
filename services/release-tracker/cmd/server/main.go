package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/adapters/eventbus/rabbitmq"
	githubclient "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/adapters/github"
	subscriptionhttp "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/adapters/subscriptions/http"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/database"
	repositorypostgresql "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/persistence/postgresql"
	releasetrackerhttp "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/transport/http"
	releasetrackerusecase "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/usecase"
	releasetrackerworker "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/release-tracker/internal/modules/releasetracker/worker"
	sharedrabbitmq "github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/shared/messaging/rabbitmq"
)

const configPath = "configs/config.yaml"

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "release-tracker")
	if err := run(log); err != nil {
		log.Error("fatal error", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	db, err := database.Open(ctx, cfg.Database.ConnectionString)
	if err != nil {
		return err
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("close database", "err", err)
		}
	}()
	if err := database.Migrate(ctx, db); err != nil {
		return err
	}

	githubTimeout, err := time.ParseDuration(cfg.GitHub.Timeout)
	if err != nil {
		return fmt.Errorf("parse GitHub timeout: %w", err)
	}
	subscriptionsTimeout, err := time.ParseDuration(cfg.Subscriptions.Timeout)
	if err != nil {
		return fmt.Errorf("parse subscriptions timeout: %w", err)
	}
	scanInterval, err := time.ParseDuration(cfg.Scanner.Interval)
	if err != nil {
		return fmt.Errorf("parse scan interval: %w", err)
	}

	publisher, err := rabbitmq.NewPublisher(sharedrabbitmq.Config{
		URL:      cfg.EventBus.URL,
		Exchange: cfg.EventBus.NotificationExchange,
		Queue:    cfg.EventBus.NotificationQueue,
		DLQ:      cfg.EventBus.NotificationDLQ,
	})
	if err != nil {
		return err
	}
	defer func() {
		if err := publisher.Close(); err != nil {
			log.Error("close notification publisher", "err", err)
		}
	}()

	tracker := releasetrackerusecase.New(
		log,
		repositorypostgresql.NewRepositoryStore(db),
		githubclient.NewClient(&http.Client{Timeout: githubTimeout}, cfg.GitHub.URL, cfg.GitHub.APIToken),
		subscriptionhttp.NewClient(&http.Client{Timeout: subscriptionsTimeout}, cfg.Subscriptions.URL),
		publisher,
	)
	scheduler := releasetrackerworker.NewScheduler(tracker, scanInterval)
	server := &http.Server{
		Addr:              ":" + cfg.Server.Port,
		Handler:           releasetrackerhttp.NewRouter(log, db, tracker),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go scheduler.Start(ctx)
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("HTTP server starting", "addr", server.Addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdownCtx)
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
