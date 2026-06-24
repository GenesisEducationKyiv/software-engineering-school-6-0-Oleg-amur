package config

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	Database       Database       `yaml:"database"`
	Server         Server         `yaml:"server"`
	EventBus       EventBus       `yaml:"eventBus"`
	ReleaseTracker ReleaseTracker `yaml:"releaseTracker"`
}

type Database struct {
	ConnectionString string `yaml:"connectionString" env:"DATABASE_URL"`
}

type Server struct {
	Host     string `yaml:"host"     env:"SERVER_HOST" env-default:"0.0.0.0"`
	Port     string `yaml:"port"     env:"SERVER_PORT" env-default:"8080"`
	GrpcPort string `yaml:"grpcPort" env:"GRPC_PORT"   env-default:"50051"`
}

//nolint:golines // Environment tags must remain single string literals.
type EventBus struct {
	URL                   string `yaml:"url"                       env:"EVENT_BUS_URL"`
	NotificationExchange  string `yaml:"notificationExchange"      env:"NOTIFICATION_EXCHANGE"`
	NotificationQueue     string `yaml:"notificationQueue"         env:"NOTIFICATION_QUEUE"`
	NotificationDLQ       string `yaml:"notificationDLQ"           env:"NOTIFICATION_DEAD_LETTER_QUEUE"`
	SubscriptionSagaQueue string `yaml:"subscriptionSagaQueue"     env:"SUBSCRIPTION_SAGA_QUEUE"             env-default:"subscription-service.subscription-saga"`
	SubscriptionSagaDLQ   string `yaml:"subscriptionSagaDLQ"       env:"SUBSCRIPTION_SAGA_DEAD_LETTER_QUEUE" env-default:"subscription-service.subscription-saga.dlq"`
}

type ReleaseTracker struct {
	URL     string `yaml:"url"     env:"RELEASE_TRACKER_URL"     env-default:"http://localhost:8081"`
	Timeout string `yaml:"timeout" env:"RELEASE_TRACKER_TIMEOUT" env-default:"10s"`
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
