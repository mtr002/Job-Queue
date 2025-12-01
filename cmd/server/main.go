package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/mtr002/Job-Queue/internal/api"
	"github.com/mtr002/Job-Queue/internal/db"
	"github.com/mtr002/Job-Queue/internal/grpc"
	"github.com/mtr002/Job-Queue/internal/interfaces"
	"github.com/mtr002/Job-Queue/internal/jobs"
	"github.com/mtr002/Job-Queue/internal/logger"
	"github.com/mtr002/Job-Queue/internal/metrics"
	"github.com/mtr002/Job-Queue/internal/nats"
	"github.com/mtr002/Job-Queue/internal/websocket"
)

func main() {
	const (
		port          = "8080"
		workerAddr    = "localhost:8081"
		migrationsDir = "migrations"
	)

	logger.Init("api-service")
	logger.Logger.Info().Msg("Starting API Service with gRPC and WebSocket")

	config := db.DefaultConfig()
	database, err := db.Connect(config)
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer database.Close()

	if err := db.RunMigrations(database, migrationsDir); err != nil {
		logger.Logger.Fatal().Err(err).Msg("Failed to run migrations")
	}

	store := db.NewStore(database)
	manager := jobs.NewManager(store, 3)

	var grpcClient *grpc.Client
	var natsClient *nats.Client

	useNATS := os.Getenv("USE_NATS")
	if useNATS == "true" {
		natsURL := os.Getenv("NATS_URL")
		if natsURL == "" {
			natsURL = "nats://localhost:4222"
		}
		client, err := nats.NewClient(natsURL)
		if err != nil {
			logger.Logger.Fatal().Err(err).Msg("Failed to connect to NATS")
		}
		defer client.Close()
		natsClient = client
		logger.Logger.Info().Str("url", natsURL).Msg("Using NATS for job submission")
	} else {
		client, err := grpc.NewClient(workerAddr)
		if err != nil {
			logger.Logger.Fatal().Err(err).Msg("Failed to connect to Worker Service")
		}
		defer client.Close()
		grpcClient = client
		logger.Logger.Info().Str("addr", workerAddr).Msg("Using gRPC for job submission")
	}

	hub := websocket.NewHub()
	go hub.Run()

	go startJobPoller(manager, hub)

	server := api.NewServer(manager, grpcClient, natsClient, hub, port, database)

	go func() {
		server.Start()
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Logger.Info().Msg("Shutting down gracefully...")
	logger.Logger.Info().Msg("API Service stopped")
}

func startJobPoller(manager *jobs.Manager, hub *websocket.Hub) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	lastJobs := make(map[string]string)

	for range ticker.C {
		allJobs, err := manager.GetAllJobs()
		if err != nil {
			continue
		}

		pendingCount := 0
		for _, job := range allJobs {
			if job.Status == interfaces.StatusPending || job.Status == interfaces.StatusRetrying {
				pendingCount++
			}

			lastStatus, exists := lastJobs[job.ID]
			if !exists || lastStatus != string(job.Status) {
				websocket.BroadcastJobUpdate(hub, job)
				lastJobs[job.ID] = string(job.Status)
			}
		}

		metrics.PendingJobs.Set(float64(pendingCount))
	}
}
