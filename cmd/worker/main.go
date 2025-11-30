package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"

	"github.com/mtr002/Job-Queue/internal/db"
	"github.com/mtr002/Job-Queue/internal/jobs"
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

	log.Println("Starting Worker Service with gRPC...")

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
	manager := jobs.NewManager(store, maxRetries)

	processor := &worker.DefaultJobProcessor{}
	workerPool := worker.NewPool(manager, processor, workerCount)
	workerPool.Start()

	lis, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	s := grpc.NewServer()
	proto.RegisterWorkerServiceServer(s, &workerServer{manager: manager})

	go func() {
		log.Printf("Worker Service gRPC server listening on :%s", port)
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down gracefully...")
	s.GracefulStop()
	workerPool.Stop()
	log.Println("Worker Service stopped")
}
