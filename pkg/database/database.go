package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

// Config holds database connection configuration
type Config struct {
	URL string
}

// Connect establishes a connection to the database
func Connect(cfg Config) (*sql.DB, error) {
	db, err := sql.Open("postgres", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	log.Println("Connected to database successfully")
	return db, nil
}

// MustConnect connects to the database or panics
func MustConnect(cfg Config) *sql.DB {
	db, err := Connect(cfg)
	if err != nil {
		log.Fatalf("Database connection failed: %v", err)
	}
	return db
}
