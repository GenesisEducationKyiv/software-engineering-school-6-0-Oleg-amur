package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

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

	emailClient := email.NewClient(cfg.Notifier.SMTPHost, cfg.Notifier.SMTPPort, cfg.Notifier.FromEmail)
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
