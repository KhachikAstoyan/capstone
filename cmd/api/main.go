package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/KhachikAstoyan/capstone/internal/api"
	"github.com/KhachikAstoyan/capstone/pkg/database"
	"github.com/KhachikAstoyan/capstone/pkg/migrations"
)

func main() {
	// Load configuration
	cfg, err := api.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting API server in %s environment", cfg.Environment)

	// Connect to API database
	db := database.MustConnect(database.Config{
		URL: cfg.DatabaseURL,
	})
	defer db.Close()

	// Convert to absolute path
	absPath, err := filepath.Abs(cfg.MigrationsPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path for migrations: %v", err)
	}

	// Run migrations
	log.Printf("Running migrations from: %s", absPath)
	if err := migrations.RunMigrations(db, absPath); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations completed successfully")

	// Start API server
	log.Printf("Starting API server on %s:%d", cfg.ServerHost, cfg.ServerPort)
	fmt.Printf("API server listening on http://%s:%d\n", cfg.ServerHost, cfg.ServerPort)

	// TODO: Add your API server logic here
	// Example:
	// router := setupRouter(db, cfg)
	// if err := router.Run(fmt.Sprintf("%s:%d", cfg.ServerHost, cfg.ServerPort)); err != nil {
	//     log.Fatalf("Failed to start server: %v", err)
	// }
}
