package email

import (
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/config"
	"github.com/kelseyhightower/envconfig"
)

// Config holds configuration for the email worker (RabbitMQ consumer + SMTP sender).
type Config struct {
	config.CommonConfig

	// RabbitMQURL is required to consume verification events (same broker as API).
	RabbitMQURL string `envconfig:"EMAIL_RABBITMQ_URL" required:"true"`
	// RabbitMQExchange must match the API publisher exchange.
	RabbitMQExchange string `envconfig:"EMAIL_RABBITMQ_EXCHANGE" default:"capstone.events"`
	// RabbitMQEmailVerificationRoutingKey must match the API routing key for verification messages.
	RabbitMQEmailVerificationRoutingKey string `envconfig:"EMAIL_RABBITMQ_EMAIL_VERIFICATION_ROUTING_KEY" default:"email.verification"`
	// RabbitMQQueue is the durable queue this service consumes from (bound to the exchange with the routing key above).
	RabbitMQQueue string `envconfig:"EMAIL_RABBITMQ_QUEUE" default:"capstone.email.verification"`

	// SMTP settings for outbound email delivery.
	SMTPHost     string `envconfig:"SMTP_HOST" required:"true"`
	SMTPPort     int    `envconfig:"SMTP_PORT" default:"587"`
	SMTPUsername string `envconfig:"SMTP_USERNAME" required:"true"`
	SMTPPassword string `envconfig:"SMTP_PASSWORD" required:"true"`
	// SMTPFrom is the envelope/header From address (e.g. "Capstone <no-reply@example.com>").
	SMTPFrom string `envconfig:"SMTP_FROM" required:"true"`
}

// LoadConfig loads the email service configuration from environment variables.
func LoadConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}
