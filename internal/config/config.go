package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Database     Database     `yaml:"database"`
	Server       Server       `yaml:"server"`
	Scanner      Scanner      `yaml:"scanner"`
	Notifier     Notifier     `yaml:"notifier"`
	GithubClient GithubClient `yaml:"github"`
}

type Database struct {
	ConnectionString string `yaml:"connectionString"`
}

type Server struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	GrpcPort string `yaml:"grpcPort"`
}

type Scanner struct {
	Interval string `yaml:"interval"`
}

type Notifier struct {
	SMTPHost        string `yaml:"smtpHost"`
	SMTPPort        string `yaml:"smtpPort"`
	FromEmail       string `yaml:"fromEmail"`
	ConfirmationUrl string `yaml:"confirmationUrl"`
}

type GithubClient struct {
	Timeout  string `yaml:"timeout"`
	Url      string `yaml:"url"`
	ApiToken string `yaml:"apiToken"`
}

func LoadConfig(path string) (*Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(content, &config); err != nil {
		return nil, fmt.Errorf("failed to parse yaml config: %w", err)
	}

	if envURL := os.Getenv("DATABASE_URL"); envURL != "" {
		config.Database.ConnectionString = envURL
	}

	if envHost := os.Getenv("SERVER_HOST"); envHost != "" {
		config.Server.Host = envHost
	}

	if envPort := os.Getenv("SERVER_PORT"); envPort != "" {
		config.Server.Port = envPort
	}

	if envGrpcPort := os.Getenv("SERVER_GRPC_PORT"); envGrpcPort != "" {
		config.Server.GrpcPort = envGrpcPort
	}

	if envSMTPHost := os.Getenv("SMTP_HOST"); envSMTPHost != "" {
		config.Notifier.SMTPHost = envSMTPHost
	}

	if envSMTPPort := os.Getenv("SMTP_PORT"); envSMTPPort != "" {
		config.Notifier.SMTPPort = envSMTPPort
	}

	if envGHUrl := os.Getenv("GITHUB_URL"); envGHUrl != "" {
		config.GithubClient.Url = envGHUrl
	}

	if envGHToken := os.Getenv("GITHUB_TOKEN"); envGHToken != "" {
		config.GithubClient.ApiToken = envGHToken
	}

	if envGHTimeout := os.Getenv("GITHUB_TIMEOUT"); envGHTimeout != "" {
		config.GithubClient.Timeout = envGHTimeout
	}

	return &config, nil
}
