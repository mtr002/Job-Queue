package api

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/mtr002/Job-Queue/internal/jobs"
)

// NewServer creates a new API server (following Mat Ryer's pattern)
func NewServer(manager *jobs.Manager, port string) *Server {
	logger := log.New(log.Writer(), "[API] ", log.LstdFlags)

	return &Server{
		manager: manager,
		port:    port,
		logger:  logger,
	}
}

// Server represents the HTTP server with job management capabilities
type Server struct {
	manager *jobs.Manager
	logger  *log.Logger
	port    string
}

// Start starts the HTTP server
func (s *Server) Start() {
	addr := fmt.Sprintf(":%s", s.port)
	s.logger.Printf("Starting server on %s", addr)

	// Create a new ServeMux
	mux := http.NewServeMux()

	// Add routes (since addRoutes is in the same package, we can call it directly)
	AddRoutes(mux, s.logger, s.manager)

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
