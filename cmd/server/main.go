package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mtr002/Job-Queue/internal/api"
	"github.com/mtr002/Job-Queue/internal/db"
	"github.com/mtr002/Job-Queue/internal/grpc"
	"github.com/mtr002/Job-Queue/internal/jobs"
)

func main() {
	const (
		port          = "8080"
		workerAddr    = "localhost:8081"
		migrationsDir = "migrations"
	)

	log.Println("Starting API Service with gRPC...")

	config := db.DefaultConfig()
	database, err := db.Connect(config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(database, migrationsDir); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	store := db.NewStore(database)
	manager := jobs.NewManager(store, 3)

	grpcClient, err := grpc.NewClient(workerAddr)
	if err != nil {
		log.Fatalf("Failed to connect to Worker Service: %v", err)
	}
	defer grpcClient.Close()

	server := api.NewServer(manager, grpcClient, port)

	go func() {
		server.Start()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	log.Println("API Service stopped")
}
