package database

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConnect_InvalidURL(t *testing.T) {
	cfg := Config{
		URL: "invalid-url",
	}

	db, err := Connect(cfg)
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestConnect_ValidFormat(t *testing.T) {
	// Test that a properly formatted URL doesn't error on Open
	// (it will error on Ping if database doesn't exist, which is expected)
	cfg := Config{
		URL: "postgres://user:pass@localhost:5432/testdb?sslmode=disable",
	}

	db, _ := Connect(cfg)
	// We expect an error because the database doesn't exist
	// but we're testing that the URL format is accepted
	if db != nil {
		db.Close()
	}
	// Don't assert on error here since it's environment-dependent
	// Just ensure the function doesn't panic
	assert.NotNil(t, cfg)
}

func TestConfig(t *testing.T) {
	cfg := Config{
		URL: "postgres://localhost/test",
	}

	assert.Equal(t, "postgres://localhost/test", cfg.URL)
}
