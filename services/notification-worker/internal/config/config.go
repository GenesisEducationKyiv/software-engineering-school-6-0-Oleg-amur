package config

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	Notifier Notifier `yaml:"notifier"`
	EventBus EventBus `yaml:"eventBus"`
}

type Notifier struct {
	SMTPHost  string `yaml:"smtpHost"  env:"SMTP_HOST"`
	SMTPPort  string `yaml:"smtpPort"  env:"SMTP_PORT"`
	FromEmail string `yaml:"fromEmail" env:"FROM_EMAIL"`
	BaseUrl   string `yaml:"baseUrl"   env:"BASE_URL"`
	Timeout   string `yaml:"timeout"   env:"SMTP_TIMEOUT" env-default:"10s"`
}

type EventBus struct {
	URL                  string `yaml:"url"                  env:"EVENT_BUS_URL"`
	NotificationExchange string `yaml:"notificationExchange" env:"NOTIFICATION_EXCHANGE"`
	NotificationQueue    string `yaml:"notificationQueue"    env:"NOTIFICATION_QUEUE"`
	NotificationDLQ      string `yaml:"notificationDLQ"      env:"NOTIFICATION_DEAD_LETTER_QUEUE"`
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
