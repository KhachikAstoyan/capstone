package config

// CommonConfig holds configuration shared across all apps
// Only put truly shared config here (environment, logging, etc.)
type CommonConfig struct {
	Environment string `envconfig:"ENVIRONMENT" default:"development"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`
}
