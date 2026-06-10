package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/notification-worker/internal/client/email"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/notification-worker/internal/config"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/notification-worker/internal/eventbus/rabbitmq"
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/notification-worker/internal/service"
)

const configPath = "configs/config.yaml"

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "notification-worker")
	slog.SetDefault(log)

	if err := runWorker(log); err != nil {
		log.Error("fatal error", "error", err)
		os.Exit(1)
	}
}

func runWorker(log *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config from %s: %w", configPath, err)
	}

	emailClient, err := setupEmailClient(cfg.Notifier)
	if err != nil {
		return err
	}
	msgBuilder := email.NewSimpleMessageBuilder(cfg.Notifier.BaseUrl)
	notificationService := service.NewNotificationService(log, emailClient, msgBuilder)

	consumer, err := rabbitmq.NewNotificationConsumer(log, cfg.EventBus.RabbitMQConfig())
	if err != nil {
		return err
	}
	defer func() {
		if err := consumer.Close(); err != nil {
			log.Error("unable to close notification consumer", "error", err)
		}
	}()

	return consumer.Subscribe(ctx, notificationService)
}

func setupEmailClient(cfg config.Notifier) (*email.Client, error) {
	timeout, err := time.ParseDuration(cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("failed to parse smtp timeout: %w", err)
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("smtp timeout must be positive")
	}

	return email.NewClient(cfg.SMTPHost, cfg.SMTPPort, cfg.FromEmail, timeout), nil
}
