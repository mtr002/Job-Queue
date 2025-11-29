package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/mtr002/Job-Queue/internal/jobs"
)

// addRoutes maps the entire API surface
func AddRoutes(
	mux *http.ServeMux,
	logger *log.Logger,
	manager *jobs.Manager,
) {
	// Register all routes
	mux.HandleFunc("/jobs", handleJobs(logger, manager))
	mux.HandleFunc("/jobs/", handleJobByID(logger, manager))

	// Catch-all route for unmatched routes
	mux.HandleFunc("/", http.NotFound)
}

// handleJobs is a generic handler for all job-related endpoints
func handleJobs(logger *log.Logger, manager *jobs.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Printf("Received request: %s %s", r.Method, r.URL.Path)
		switch r.Method {
		case http.MethodGet:
			handleListJobs(w, r, logger, manager)
		case http.MethodPost:
			handleCreateJob(w, r, logger, manager)
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

// handleCreateJob is a handler for the /jobs endpoint
func handleCreateJob(w http.ResponseWriter, r *http.Request, logger *log.Logger, manager *jobs.Manager) {
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

	// Decode the request
	var req JobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		http.Error(w, "Job type is required", http.StatusBadRequest)
		return
	}

	job, err := manager.SubmitJob(req.Type, req.Payload)
	if err != nil {
		logger.Printf("Failed to submit job: %v", err)
		http.Error(w, "Failed to submit job: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Encode response
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
