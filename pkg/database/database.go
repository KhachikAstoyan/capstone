package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/KhachikAstoyan/capstone/pkg/logger"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// Config holds database connection configuration
type Config struct {
	URL string
}

// Connect establishes a connection to the database
func Connect(ctx context.Context, cfg Config) (*sql.DB, error) {
	log := logger.FromContext(ctx)

	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Info("Connected to database successfully")
	return db, nil
}

// MustConnect connects to the database or panics
func MustConnect(ctx context.Context, cfg Config) *sql.DB {
	log := logger.FromContext(ctx)

	db, err := Connect(ctx, cfg)
	if err != nil {
		log.Fatal("Database connection failed", zap.Error(err))
	}
	return db
}
