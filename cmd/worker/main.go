package main

import (
	"context"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/mtr002/Job-Queue/internal/db"
	"github.com/mtr002/Job-Queue/internal/jobs"
	"github.com/mtr002/Job-Queue/internal/logger"
	"github.com/mtr002/Job-Queue/internal/nats"
	"github.com/mtr002/Job-Queue/internal/worker"
	"github.com/mtr002/Job-Queue/proto"
)

type workerServer struct {
	proto.UnimplementedWorkerServiceServer
	manager *jobs.Manager
}

func (s *workerServer) SubmitJob(ctx context.Context, req *proto.SubmitJobRequest) (*proto.SubmitJobResponse, error) {
	maxAttempts := int(req.MaxAttempts)
	if maxAttempts <= 0 {
		maxAttempts = 3
	}

	job, err := s.manager.SubmitJob(req.Type, req.Payload)
	if err != nil {
		return nil, err
	}

	return &proto.SubmitJobResponse{
		JobId:     job.ID,
		Status:    string(job.Status),
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
	}, nil
}

func (s *workerServer) GetJobStatus(ctx context.Context, req *proto.GetJobRequest) (*proto.JobStatusResponse, error) {
	job, err := s.manager.GetJob(req.JobId)
	if err != nil {
		return nil, err
	}

	resp := &proto.JobStatusResponse{
		JobId:       job.ID,
		Type:        job.Type,
		Status:      string(job.Status),
		Payload:     job.Payload,
		Result:      job.Result,
		Error:       job.Error,
		Attempts:    int32(job.Attempts),
		MaxAttempts: int32(job.MaxAttempts),
		CreatedAt:   job.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   job.UpdatedAt.Format(time.RFC3339),
	}

	return resp, nil
}

func (s *workerServer) NotifyJobCompleted(ctx context.Context, req *proto.ProcessJobRequest) (*proto.ProcessJobResponse, error) {
	job, err := s.manager.GetJob(req.JobId)
	if err != nil {
		return &proto.ProcessJobResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if err := s.manager.UpdateJobCompleted(job, "completed"); err != nil {
		return &proto.ProcessJobResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &proto.ProcessJobResponse{
		Success: true,
		Message: "Job marked as completed",
	}, nil
}

func (s *workerServer) NotifyJobFailed(ctx context.Context, req *proto.ProcessJobRequest) (*proto.ProcessJobResponse, error) {
	job, err := s.manager.GetJob(req.JobId)
	if err != nil {
		return &proto.ProcessJobResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	if err := s.manager.UpdateJobFailed(job, "job processing failed"); err != nil {
		return &proto.ProcessJobResponse{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	return &proto.ProcessJobResponse{
		Success: true,
		Message: "Job marked as failed",
	}, nil
}

func main() {
	const (
		port          = "8081"
		workerCount   = 3
		maxRetries    = 3
		migrationsDir = "migrations"
	)

	logger.Init("worker-service")
	logger.Logger.Info().Msg("Starting Worker Service with gRPC")

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
	manager := jobs.NewManager(store, maxRetries)

	processor := &worker.DefaultJobProcessor{}
	workerPool := worker.NewPool(manager, processor, workerCount)
	workerPool.Start()

	var natsServer *nats.Server
	useNATS := os.Getenv("USE_NATS")
	if useNATS == "true" {
		natsURL := os.Getenv("NATS_URL")
		if natsURL == "" {
			natsURL = "nats://localhost:4222"
		}
		server, err := nats.NewServer(natsURL, manager)
		if err != nil {
			logger.Logger.Fatal().Err(err).Msg("Failed to create NATS server")
		}
		if err := server.Subscribe(); err != nil {
			logger.Logger.Fatal().Err(err).Msg("Failed to subscribe to NATS")
		}
		defer server.Close()
		natsServer = server
		logger.Logger.Info().Str("url", natsURL).Msg("NATS consumer started")
	}

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		logger.Logger.Fatal().Err(err).Msg("Failed to listen")
	}

	s := grpc.NewServer()
	proto.RegisterWorkerServiceServer(s, &workerServer{manager: manager})

	go func() {
		logger.Logger.Info().Str("port", port).Msg("Worker Service gRPC server listening")
		if err := s.Serve(lis); err != nil {
			logger.Logger.Fatal().Err(err).Msg("Failed to serve")
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Logger.Info().Msg("Shutting down gracefully...")
	s.GracefulStop()
	workerPool.Stop()
	if natsServer != nil {
		natsServer.Close()
	}
	logger.Logger.Info().Msg("Worker Service stopped")
}
