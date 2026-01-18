package worker

import (
	"fmt"

	"github.com/KhachikAstoyan/capstone/internal/config"
	"github.com/kelseyhightower/envconfig"
)

// Config holds all configuration for the Worker
type Config struct {
	config.CommonConfig
}

// LoadConfig loads the Worker configuration from environment variables
func LoadConfig() (*Config, error) {
	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}
	return &cfg, nil
}
