package logger

import (
	"context"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type contextKey string

const loggerKey contextKey = "logger"

// Init creates and returns a new logger based on environment
// environment should be "development", "production", or "staging"
func Init(environment string) *zap.Logger {
	var logger *zap.Logger
	var err error

	switch environment {
	case "development", "dev":
		// Development logger with colors and human-readable output
		config := zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logger, err = config.Build()
	case "production", "prod", "staging":
		logger, err = zap.NewProduction()
	default:
		logger, err = zap.NewProduction()
	}

	if err != nil {
		panic(err)
	}

	return logger
}

// WithLogger adds logger to context
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

// FromContext retrieves logger from context
func FromContext(ctx context.Context) *zap.Logger {
	if logger, ok := ctx.Value(loggerKey).(*zap.Logger); ok {
		return logger
	}
	// Return a no-op logger if not found
	return zap.NewNop()
}
