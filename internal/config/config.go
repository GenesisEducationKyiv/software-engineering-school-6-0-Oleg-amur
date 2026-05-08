package config

import (
	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	Database     Database     `yaml:"database"`
	Server       Server       `yaml:"server"`
	Scanner      Scanner      `yaml:"scanner"`
	Notifier     Notifier     `yaml:"notifier"`
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

type Notifier struct {
	SMTPHost        string `yaml:"smtpHost"        env:"SMTP_HOST"`
	SMTPPort        string `yaml:"smtpPort"        env:"SMTP_PORT"`
	FromEmail       string `yaml:"fromEmail"       env:"FROM_EMAIL"`
	ConfirmationUrl string `yaml:"confirmationUrl" env:"CONFIRMATION_URL"`
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
