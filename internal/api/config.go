package api

import (
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/config"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the API server
type Config struct {
	config.CommonConfig

	// API-specific configuration
	ServerPort     int    `envconfig:"API_PORT" default:"8080"`
	ServerHost     string `envconfig:"API_HOST" default:"0.0.0.0"`
	AllowedOrigins string `envconfig:"API_ALLOWED_ORIGINS" default:"*"`

	DatabaseURL    string `envconfig:"API_DATABASE_URL" default:"postgres://postgres:postgres@localhost:5432/api?sslmode=disable"`
	MigrationsPath string `envconfig:"API_MIGRATIONS_PATH" default:"./internal/api/migrations"`
}

// LoadConfig loads the API configuration from environment variables
func LoadConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}
