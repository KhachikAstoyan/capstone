package logger

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestWithLogger(t *testing.T) {
	log := zap.NewNop()
	ctx := context.Background()

	// Add logger to context
	ctx = WithLogger(ctx, log)

	// Retrieve logger from context
	retrieved := FromContext(ctx)

	assert.NotNil(t, retrieved)
	assert.Equal(t, log, retrieved)
}

func TestFromContext_NoLogger(t *testing.T) {
	ctx := context.Background()

	// Should return no-op logger when not in context
	log := FromContext(ctx)

	assert.NotNil(t, log)
	// Verify it's a no-op logger by checking it doesn't panic
	log.Info("This should not panic")
}

func TestInit_Development(t *testing.T) {
	log := Init("development")

	assert.NotNil(t, log)
	// Verify we can log without panic
	log.Info("Test log in development")
}

func TestInit_Production(t *testing.T) {
	log := Init("production")

	assert.NotNil(t, log)
	// Verify we can log without panic
	log.Info("Test log in production")
}

func TestInit_Default(t *testing.T) {
	// Unknown environment should default to production
	log := Init("unknown")

	assert.NotNil(t, log)
	log.Info("Test log with unknown environment")
}
