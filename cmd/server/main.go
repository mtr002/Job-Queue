package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mtr002/Job-Queue/internal/api"
	"github.com/mtr002/Job-Queue/internal/db"
	"github.com/mtr002/Job-Queue/internal/jobs"
	"github.com/mtr002/Job-Queue/internal/worker"
)

func main() {
	// Configuration
	const (
		port          = "8080"
		workerCount   = 3
		maxRetries    = 3
		migrationsDir = "migrations"
	)

	log.Println("Starting JobQueue server with PostgreSQL persistence...")

	// Connect to database
	config := db.DefaultConfig()
	database, err := db.Connect(config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	// Run migrations
	if err := db.RunMigrations(database, migrationsDir); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Create database store
	store := db.NewStore(database)

	// Create job manager with database persistence
	manager := jobs.NewManager(store, maxRetries)

	// Create and start worker pool
	processor := &worker.DefaultJobProcessor{}
	workerPool := worker.NewPool(manager, processor, workerCount)
	workerPool.Start()

	// Create and start API server
	server := api.NewServer(manager, port)

	// Handle graceful shutdown
	go func() {
		server.Start()
	}()

	// Wait for interrupt signal to gracefully shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	workerPool.Stop()
	log.Println("Server stopped")
}
