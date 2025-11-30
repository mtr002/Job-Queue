package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/mtr002/Job-Queue/internal/grpc"
	"github.com/mtr002/Job-Queue/internal/interfaces"
	"github.com/mtr002/Job-Queue/internal/jobs"
	"github.com/mtr002/Job-Queue/internal/logger"
	"github.com/mtr002/Job-Queue/internal/metrics"
	"github.com/mtr002/Job-Queue/internal/nats"
	"github.com/mtr002/Job-Queue/internal/websocket"
)

func AddRoutes(
	mux *http.ServeMux,
	manager *jobs.Manager,
	grpcClient *grpc.Client,
	natsClient *nats.Client,
	hub *websocket.Hub,
) {
	mux.HandleFunc("/jobs", correlationMiddleware(handleJobs(manager, grpcClient, natsClient, hub)))
	mux.HandleFunc("/jobs/", correlationMiddleware(handleJobByID(manager)))
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		websocket.HandleWebSocket(hub, w, r)
	})
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health", HandleHealth)
	mux.HandleFunc("/health/ready", HandleReadiness)
	mux.HandleFunc("/health/live", HandleLiveness)
	mux.HandleFunc("/", handleRoot(hub))
}

func correlationMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		correlationID := r.Header.Get("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}
		ctx := r.Context()
		ctx = context.WithValue(ctx, "correlation_id", correlationID)
		r = r.WithContext(ctx)
		next(w, r)
	}
}

func handleRoot(hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "web/dashboard.html")
			return
		}
		http.NotFound(w, r)
	}
}

func handleJobs(manager *jobs.Manager, grpcClient *grpc.Client, natsClient *nats.Client, hub *websocket.Hub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		correlationID := getCorrelationID(r.Context())
		log := logger.WithCorrelationID(correlationID)
		log.Info().
			Str("method", r.Method).
			Str("path", r.URL.Path).
			Msg("Received request")

		switch r.Method {
		case http.MethodGet:
			handleListJobs(w, r, manager, correlationID)
		case http.MethodPost:
			handleCreateJob(w, r, manager, grpcClient, natsClient, hub, correlationID)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleJobByID(manager *jobs.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/jobs/")
		if path == "" {
			http.Error(w, "Job ID is required", http.StatusBadRequest)
			return
		}

		correlationID := getCorrelationID(r.Context())
		handleGetJob(w, r, path, manager, correlationID)
	}
}

func handleCreateJob(w http.ResponseWriter, r *http.Request, manager *jobs.Manager, grpcClient *grpc.Client, natsClient *nats.Client, hub *websocket.Hub, correlationID string) {
	type JobRequest struct {
		Type    string `json:"type"`
		Payload string `json:"payload"`
	}

	type JobResponse struct {
		ID        string `json:"id"`
		Type      string `json:"type"`
		Status    string `json:"status"`
		Payload   string `json:"payload"`
		CreatedAt string `json:"created_at"`
		UpdatedAt string `json:"updated_at"`
		Error     string `json:"error"`
		Result    string `json:"result"`
	}

	log := logger.WithCorrelationID(correlationID)

	var req JobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Error().Err(err).Msg("Invalid JSON request")
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		log.Warn().Msg("Job type missing")
		http.Error(w, "Job type is required", http.StatusBadRequest)
		return
	}

	var job *interfaces.Job
	var err error

	if natsClient != nil {
		msg := &nats.JobSubmissionMessage{
			Type:        req.Type,
			Payload:     req.Payload,
			MaxAttempts: 3,
		}
		if err := natsClient.PublishJobSubmission(msg); err != nil {
			log.Error().Err(err).Msg("Failed to submit job via NATS")
			http.Error(w, "Failed to submit job: "+err.Error(), http.StatusInternalServerError)
			return
		}
		job, err = manager.SubmitJob(req.Type, req.Payload)
		if err != nil {
			log.Error().Err(err).Msg("Failed to create job in database")
			http.Error(w, "Failed to submit job: "+err.Error(), http.StatusInternalServerError)
			return
		}
		log.Info().Str("job_id", job.ID).Msg("Job submitted via NATS")
	} else if grpcClient != nil {
		job, err = grpcClient.SubmitJob(req.Type, req.Payload, 3)
		if err != nil {
			log.Error().Err(err).Msg("Failed to submit job via gRPC")
			http.Error(w, "Failed to submit job: "+err.Error(), http.StatusInternalServerError)
			return
		}
		metrics.JobsSubmittedTotal.Inc()
	} else {
		http.Error(w, "No job submission method available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	response := JobResponse{
		ID:        job.ID,
		Type:      job.Type,
		Status:    string(job.Status),
		Payload:   job.Payload,
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
		UpdatedAt: job.UpdatedAt.Format(time.RFC3339),
		Error:     job.Error,
		Result:    job.Result,
	}

	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
		return
	}

	log.Info().Str("job_id", job.ID).Msg("Job submitted successfully")
	websocket.BroadcastJobUpdate(hub, job)
}

func handleGetJob(w http.ResponseWriter, _ *http.Request, jobID string, manager *jobs.Manager, correlationID string) {
	log := logger.WithCorrelationID(correlationID)

	job, err := manager.GetJob(jobID)
	if err != nil {
		log.Warn().Str("job_id", jobID).Msg("Job not found")
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(job); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
	}
}

func handleListJobs(w http.ResponseWriter, _ *http.Request, manager *jobs.Manager, correlationID string) {
	log := logger.WithCorrelationID(correlationID)

	jobs, err := manager.GetAllJobs()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get all jobs")
		http.Error(w, "Failed to retrieve jobs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	}); err != nil {
		log.Error().Err(err).Msg("Failed to encode response")
	}
}

func getCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value("correlation_id").(string); ok {
		return id
	}
	return ""
}
