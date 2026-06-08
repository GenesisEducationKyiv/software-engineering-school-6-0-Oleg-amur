package config

import (
	"github.com/GenesisEducationKyiv/software-engineering-school-6-0-Oleg-amur/services/notification-worker/internal/eventbus/rabbitmq"
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Notifier Notifier `yaml:"notifier"`
	EventBus EventBus `yaml:"eventBus"`
}

type Notifier struct {
	SMTPHost  string `yaml:"smtpHost"  env:"SMTP_HOST"`
	SMTPPort  string `yaml:"smtpPort"  env:"SMTP_PORT"`
	FromEmail string `yaml:"fromEmail" env:"FROM_EMAIL"`
	BaseUrl   string `yaml:"baseUrl"   env:"BASE_URL"`
}

type EventBus struct {
	URL                  string `yaml:"url"                  env:"EVENT_BUS_URL"                  env-default:"amqp://guest:guest@localhost:5672/"`
	NotificationExchange string `yaml:"notificationExchange" env:"NOTIFICATION_EXCHANGE"          env-default:"notifications"`
	NotificationQueue    string `yaml:"notificationQueue"    env:"NOTIFICATION_QUEUE"             env-default:"notification-worker"`
	NotificationDLQ      string `yaml:"notificationDLQ"      env:"NOTIFICATION_DEAD_LETTER_QUEUE" env-default:"notification-worker.dlq"`
}

func (cfg EventBus) RabbitMQConfig() rabbitmq.Config {
	return rabbitmq.Config{
		URL:      cfg.URL,
		Exchange: cfg.NotificationExchange,
		Queue:    cfg.NotificationQueue,
		DLQ:      cfg.NotificationDLQ,
	}
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
