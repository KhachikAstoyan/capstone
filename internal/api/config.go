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
	SecureCookies  bool   `envconfig:"API_SECURE_COOKIES" default:"false"`

	DatabaseURL    string `envconfig:"API_DATABASE_URL" required:"true"`
	MigrationsPath string `envconfig:"API_MIGRATIONS_PATH" default:"./internal/api/migrations"`

	// JWT configuration
	JWTSecret               string `envconfig:"JWT_SECRET" required:"true"`
	JWTAccessTokenDuration  int    `envconfig:"JWT_ACCESS_TOKEN_DURATION" default:"900"`
	JWTRefreshTokenDuration int    `envconfig:"JWT_REFRESH_TOKEN_DURATION" default:"604800"`
}

// LoadConfig loads the API configuration from environment variables
func LoadConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}
