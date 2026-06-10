package config

import "github.com/ilyakaznacheev/cleanenv"

type Config struct {
	Database     Database     `yaml:"database"`
	Server       Server       `yaml:"server"`
	Scanner      Scanner      `yaml:"scanner"`
	EventBus     EventBus     `yaml:"eventBus"`
	GithubClient GithubClient `yaml:"github"`
}

type Database struct {
	ConnectionString string `yaml:"connectionString" env:"DATABASE_URL"`
}

type Server struct {
	Host     string `yaml:"host"     env:"SERVER_HOST" env-default:"0.0.0.0"`
	Port     string `yaml:"port"     env:"SERVER_PORT" env-default:"8080"`
	GrpcPort string `yaml:"grpcPort" env:"GRPC_PORT"   env-default:"50051"`
}

type Scanner struct {
	Interval string `yaml:"interval" env:"SCAN_INTERVAL" env-default:"1h"`
}

type EventBus struct {
	URL                  string `yaml:"url"                  env:"EVENT_BUS_URL"`
	NotificationExchange string `yaml:"notificationExchange" env:"NOTIFICATION_EXCHANGE"`
	NotificationQueue    string `yaml:"notificationQueue"    env:"NOTIFICATION_QUEUE"`
	NotificationDLQ      string `yaml:"notificationDLQ"      env:"NOTIFICATION_DEAD_LETTER_QUEUE"`
}

type GithubClient struct {
	Timeout  string `yaml:"timeout"  env:"GITHUB_TIMEOUT" env-default:"10s"`
	Url      string `yaml:"url"      env:"GITHUB_URL"     env-default:"https://api.github.com"`
	ApiToken string `yaml:"apiToken" env:"GITHUB_TOKEN"`
}

func LoadConfig(path string) (*Config, error) {
	var cfg Config

	if err := cleanenv.ReadConfig(path, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
