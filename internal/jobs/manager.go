package jobs

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Manager handles job storage and queueing
type Manager struct {
	jobs     map[string]*Job // In-memory storage for jobs
	jobQueue chan *Job       // Channel-based job queue
	mu       sync.RWMutex
	workers  int // Number of worker goroutines
}

// NewManager creates a new job manager
func NewManager(workers, queueSize int) *Manager {
	return &Manager{
		jobs:     make(map[string]*Job),
		jobQueue: make(chan *Job, queueSize),
		workers:  workers,
	}
}

// SubmitJob creates a new job and adds it to the queue
func (m *Manager) SubmitJob(jobType, payload string) (*Job, error) {
	if jobType == "" {
		return nil, fmt.Errorf("job type cannot be empty")
	}

	job := &Job{
		ID:        uuid.New().String(),
		Type:      jobType,
		Payload:   payload,
		Status:    StatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.mu.Lock()
	m.jobs[job.ID] = job
	m.mu.Unlock()

	// Send job to queue (non-blocking)
	select {
	case m.jobQueue <- job:
		// Job successfully queued
	default:
		// Queue is full, update status to failed
		if err := m.updateJobStatus(job.ID, StatusFailed, "", "queue is full"); err != nil {
			log.Printf("Failed to update job status: %v", err)
		}
		return job, fmt.Errorf("job queue is full")
	}

	return job, nil
}

// GetJob retrieves a job by ID
func (m *Manager) GetJob(id string) (*Job, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	job, exists := m.jobs[id]
	if !exists {
		return nil, fmt.Errorf("job with ID %s not found", id)
	}

	// Return a copy to avoid external modifications
	jobCopy := *job
	return &jobCopy, nil
}

// GetAllJobs returns all jobs (useful for debugging/monitoring)
func (m *Manager) GetAllJobs() []*Job {
	m.mu.RLock()
	defer m.mu.RUnlock()

	jobs := make([]*Job, 0, len(m.jobs))
	for _, job := range m.jobs {
		jobCopy := *job
		jobs = append(jobs, &jobCopy)
	}
	return jobs
}

// GetJobQueue returns the job queue channel (for workers)
func (m *Manager) GetJobQueue() <-chan *Job {
	return m.jobQueue
}

// updateJobStatus updates a job's status, result, and error
func (m *Manager) updateJobStatus(id string, status JobStatus, result, errorMsg string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	job, exists := m.jobs[id]
	if !exists {
		return fmt.Errorf("job with ID %s not found", id)
	}

	job.Status = status
	job.UpdatedAt = time.Now()
	if result != "" {
		job.Result = result
	}
	if errorMsg != "" {
		job.Error = errorMsg
	}

	return nil
}

// UpdateJobProcessing marks a job as processing
func (m *Manager) UpdateJobProcessing(id string) error {
	return m.updateJobStatus(id, StatusProcessing, "", "")
}

// UpdateJobCompleted marks a job as completed with result
func (m *Manager) UpdateJobCompleted(id, result string) error {
	return m.updateJobStatus(id, StatusCompleted, result, "")
}

// UpdateJobFailed marks a job as failed with error
func (m *Manager) UpdateJobFailed(id, errorMsg string) error {
	return m.updateJobStatus(id, StatusFailed, "", errorMsg)
}

// Close shuts down the job manager
func (m *Manager) Close() {
	close(m.jobQueue)
}
