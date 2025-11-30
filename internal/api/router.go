package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mtr002/Job-Queue/internal/grpc"
	"github.com/mtr002/Job-Queue/internal/jobs"
)

func AddRoutes(
	mux *http.ServeMux,
	logger *log.Logger,
	manager *jobs.Manager,
	grpcClient *grpc.Client,
) {
	mux.HandleFunc("/jobs", handleJobs(logger, manager, grpcClient))
	mux.HandleFunc("/jobs/", handleJobByID(logger, manager))
	mux.HandleFunc("/", http.NotFound)
}

func handleJobs(logger *log.Logger, manager *jobs.Manager, grpcClient *grpc.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("Received request: %s %s", r.Method, r.URL.Path)
		switch r.Method {
		case http.MethodGet:
			handleListJobs(w, r, logger, manager)
		case http.MethodPost:
			handleCreateJob(w, r, logger, grpcClient)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// handleJobByID is a handler for the /jobs/:id endpoint
func handleJobByID(logger *log.Logger, manager *jobs.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// Extract job ID from URL
		path := strings.TrimPrefix(r.URL.Path, "/jobs/")
		if path == "" {
			http.Error(w, "Job ID is required", http.StatusBadRequest)
			return
		}

		handleGetJob(w, r, path, logger, manager)
	}
}

func handleCreateJob(w http.ResponseWriter, r *http.Request, logger *log.Logger, grpcClient *grpc.Client) {
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

	var req JobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		http.Error(w, "Job type is required", http.StatusBadRequest)
		return
	}

	job, err := grpcClient.SubmitJob(req.Type, req.Payload, 3)
	if err != nil {
		logger.Printf("Failed to submit job via gRPC: %v", err)
		http.Error(w, "Failed to submit job: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(JobResponse{
		ID:        job.ID,
		Type:      job.Type,
		Status:    string(job.Status),
		Payload:   job.Payload,
		CreatedAt: job.CreatedAt.Format(time.RFC3339),
		UpdatedAt: job.UpdatedAt.Format(time.RFC3339),
		Error:     job.Error,
		Result:    job.Result,
	}); err != nil {
		logger.Printf("Failed to encode response: %v", err)
	}
}

func handleGetJob(w http.ResponseWriter, _ *http.Request, jobID string, logger *log.Logger, manager *jobs.Manager) {
	job, err := manager.GetJob(jobID)
	if err != nil {
		http.Error(w, "Job not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(job); err != nil {
		logger.Printf("Failed to encode response: %v", err)
	}
}

func handleListJobs(w http.ResponseWriter, _ *http.Request, logger *log.Logger, manager *jobs.Manager) {
	jobs, err := manager.GetAllJobs()
	if err != nil {
		logger.Printf("Failed to get all jobs: %v", err)
		http.Error(w, "Failed to retrieve jobs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]interface{}{
		"jobs":  jobs,
		"count": len(jobs),
	}); err != nil {
		logger.Printf("Failed to encode response: %v", err)
	}
}
