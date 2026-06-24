package config

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	Database      Database      `yaml:"database"`
	Server        Server        `yaml:"server"`
	Scanner       Scanner       `yaml:"scanner"`
	Subscriptions Subscriptions `yaml:"subscriptions"`
	EventBus      EventBus      `yaml:"eventBus"`
	GitHub        GitHub        `yaml:"github"`
}

type Database struct {
	ConnectionString string `yaml:"connectionString" env:"DATABASE_URL"`
}

type Server struct {
	Host string `yaml:"host" env:"SERVER_HOST" env-default:"0.0.0.0"`
	Port string `yaml:"port" env:"SERVER_PORT" env-default:"8081"`
}

type Scanner struct {
	Interval string `yaml:"interval" env:"SCAN_INTERVAL" env-default:"1h"`
}

//nolint:golines // Environment tags must remain single string literals.
type Subscriptions struct {
	URL     string `yaml:"url" env:"SUBSCRIPTIONS_URL" env-default:"http://localhost:8080"`
	Timeout string `yaml:"timeout" env:"SUBSCRIPTIONS_TIMEOUT" env-default:"10s"`
}

type EventBus struct {
	URL                  string `yaml:"url"                  env:"EVENT_BUS_URL"`
	NotificationExchange string `yaml:"notificationExchange" env:"NOTIFICATION_EXCHANGE"`
	NotificationQueue    string `yaml:"notificationQueue"    env:"NOTIFICATION_QUEUE"`
	NotificationDLQ      string `yaml:"notificationDLQ"      env:"NOTIFICATION_DEAD_LETTER_QUEUE"`
}

type GitHub struct {
	URL      string `yaml:"url"      env:"GITHUB_URL"     env-default:"https://api.github.com"`
	APIToken string `yaml:"apiToken" env:"GITHUB_TOKEN"`
	Timeout  string `yaml:"timeout"  env:"GITHUB_TIMEOUT" env-default:"10s"`
}

func Load(path string) (*Config, error) {
	var cfg Config
	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
