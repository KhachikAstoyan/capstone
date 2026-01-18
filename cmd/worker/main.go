package main

import (
	"log"

	_ "github.com/lib/pq"

	"github.com/KhachikAstoyan/capstone/internal/worker"
)

func main() {
	// Load configuration
	cfg, err := worker.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	log.Printf("Starting worker in %s environment", cfg.Environment)
}
