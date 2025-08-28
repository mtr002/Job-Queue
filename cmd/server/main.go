package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/mtr002/Job-Queue/internal/api"
	"github.com/mtr002/Job-Queue/internal/jobs"
	"github.com/mtr002/Job-Queue/internal/worker"
)

func main() {
	// Configuration
	const (
		port        = "8080"
		workerCount = 3
		queueSize   = 100
	)

	log.Println("Starting JobQueue server...")

	// Create job manager
	manager := jobs.NewManager(workerCount, queueSize)

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
	manager.Close()
	log.Println("Server stopped")
}
