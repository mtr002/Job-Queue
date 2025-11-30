package api

import (
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/mtr002/Job-Queue/internal/grpc"
	"github.com/mtr002/Job-Queue/internal/jobs"
	"github.com/mtr002/Job-Queue/internal/logger"
	"github.com/mtr002/Job-Queue/internal/nats"
	"github.com/mtr002/Job-Queue/internal/websocket"
)

func NewServer(manager *jobs.Manager, grpcClient *grpc.Client, natsClient *nats.Client, hub *websocket.Hub, port string, database *sql.DB) *Server {
	SetDBConnection(database)
	return &Server{
		manager:    manager,
		grpcClient: grpcClient,
		natsClient: natsClient,
		hub:        hub,
		port:       port,
	}
}

type Server struct {
	manager    *jobs.Manager
	grpcClient *grpc.Client
	natsClient *nats.Client
	hub        *websocket.Hub
	port       string
}

func (s *Server) Start() {
	addr := fmt.Sprintf(":%s", s.port)
	logger.Logger.Info().Str("addr", addr).Msg("Starting server")

	mux := http.NewServeMux()
	AddRoutes(mux, s.manager, s.grpcClient, s.natsClient, s.hub)

	server := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	if err := server.ListenAndServe(); err != nil {
		logger.Logger.Fatal().Err(err).Msg("Failed to start server")
	}
}
