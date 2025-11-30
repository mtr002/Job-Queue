package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mtr002/Job-Queue/internal/grpc"
	"github.com/mtr002/Job-Queue/internal/jobs"
	"github.com/mtr002/Job-Queue/internal/websocket"
)

func NewServer(manager *jobs.Manager, grpcClient *grpc.Client, hub *websocket.Hub, port string) *Server {
	logger := log.New(log.Writer(), "[API] ", log.LstdFlags)

	return &Server{
		manager:    manager,
		grpcClient: grpcClient,
		hub:        hub,
		port:       port,
		logger:     logger,
	}
}

type Server struct {
	manager    *jobs.Manager
	grpcClient *grpc.Client
	hub        *websocket.Hub
	logger     *log.Logger
	port       string
}

func (s *Server) Start() {
	addr := fmt.Sprintf(":%s", s.port)
	s.logger.Printf("Starting server on %s", addr)

	mux := http.NewServeMux()
	AddRoutes(mux, s.logger, s.manager, s.grpcClient, s.hub)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		s.logger.Fatalf("Failed to start server: %v", err)
	}
}
