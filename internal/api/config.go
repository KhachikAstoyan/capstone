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

	// FrontendURL is the SPA origin (no trailing path). Used to build email verification links: {FrontendURL}?token=...
	FrontendURL string `envconfig:"API_FRONTEND_URL" default:"http://localhost:5173"`

	// RabbitMQURL enables publishing email verification events. If empty, publishing is a no-op.
	RabbitMQURL string `envconfig:"API_RABBITMQ_URL"`
	// RabbitMQExchange is the durable topic exchange used for app events (declared on connect).
	RabbitMQExchange string `envconfig:"API_RABBITMQ_EXCHANGE" default:"capstone.events"`
	// RabbitMQEmailVerificationRoutingKey is the routing key for verification messages.
	RabbitMQEmailVerificationRoutingKey string `envconfig:"API_RABBITMQ_EMAIL_VERIFICATION_ROUTING_KEY" default:"email.verification"`

	DatabaseURL    string `envconfig:"API_DATABASE_URL" required:"true"`
	MigrationsPath string `envconfig:"API_MIGRATIONS_PATH" default:"./internal/api/migrations"`

	// ControlPlaneURL is the base URL of the execution control plane service.
	ControlPlaneURL string `envconfig:"API_CONTROL_PLANE_URL" required:"true"`
	// ControlPlaneKey is the shared secret for X-Internal-Key auth. Empty = dev mode.
	ControlPlaneKey string `envconfig:"API_CONTROL_PLANE_KEY"`

	// JWT configuration
	JWTSecret               string `envconfig:"JWT_SECRET" required:"true"`
	JWTAccessTokenDuration  int    `envconfig:"JWT_ACCESS_TOKEN_DURATION" default:"900"`
	JWTRefreshTokenDuration int    `envconfig:"JWT_REFRESH_TOKEN_DURATION" default:"604800"`

	// AI validation configuration
	AIProvider   string `envconfig:"AI_PROVIDER" default:"anthropic"`
	AIModel      string `envconfig:"AI_MODEL" default:"claude-opus-4-1"`
	AIAPIKey     string `envconfig:"AI_API_KEY" required:"true"`
	AIAPIBaseURL string `envconfig:"OPENAI_BASE_URL"`
}

// LoadConfig loads the API configuration from environment variables
func LoadConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}
