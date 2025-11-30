package jobs

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/mtr002/Job-Queue/internal/interfaces"
	"github.com/mtr002/Job-Queue/internal/logger"
	"github.com/mtr002/Job-Queue/internal/metrics"
)

// Manager handles job storage and queueing with database persistence
type Manager struct {
	store             interfaces.JobStore
	defaultMaxRetries int
}

// NewManager creates a new job manager with database persistence
func NewManager(store interfaces.JobStore, defaultMaxRetries int) *Manager {
	if defaultMaxRetries <= 0 {
		defaultMaxRetries = 3 // Default to 3 retries
	}

	return &Manager{
		store:             store,
		defaultMaxRetries: defaultMaxRetries,
	}
}

// SubmitJob creates a new job and persists it to the database
func (m *Manager) SubmitJob(jobType, payload string) (*interfaces.Job, error) {
	if jobType == "" {
		return nil, fmt.Errorf("job type cannot be empty")
	}

	job := &interfaces.Job{
		ID:          uuid.New().String(),
		Type:        jobType,
		Payload:     payload,
		Status:      interfaces.StatusPending,
		Attempts:    0,
		MaxAttempts: m.defaultMaxRetries,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := m.store.CreateJob(job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	metrics.JobsSubmittedTotal.Inc()
	log := logger.WithJobID(job.ID)
	log.Info().Str("type", job.Type).Msg("Job submitted successfully")
	return job, nil
}

// GetJob retrieves a job by ID from the database
func (m *Manager) GetJob(id string) (*interfaces.Job, error) {
	return m.store.GetJob(id)
}

// GetAllJobs returns all jobs from the database
func (m *Manager) GetAllJobs() ([]*interfaces.Job, error) {
	return m.store.GetAllJobs()
}

// GetPendingJob retrieves the next pending job for processing
func (m *Manager) GetPendingJob() (*interfaces.Job, error) {
	return m.store.GetPendingJob()
}

// UpdateJobCompleted marks a job as completed with result
func (m *Manager) UpdateJobCompleted(job *interfaces.Job, result string) error {
	job.Status = interfaces.StatusCompleted
	job.Result = result
	job.UpdatedAt = time.Now()

	if err := m.store.UpdateJob(job); err != nil {
		return fmt.Errorf("failed to update job as completed: %w", err)
	}

	metrics.JobsCompletedTotal.Inc()
	log := logger.WithJobID(job.ID)
	log.Info().Msg("Job completed successfully")
	return nil
}

// UpdateJobFailed handles job failure and implements retry logic
func (m *Manager) UpdateJobFailed(job *interfaces.Job, errorMsg string) error {
	job.Error = errorMsg
	job.IncrementAttempts()
	job.UpdatedAt = time.Now()

	if job.CanRetry() {
		// Job can be retried - set it to retrying status with backoff
		job.Status = interfaces.StatusRetrying
		job.SetRetryAfter(1) // 1 second base delay

		log := logger.WithJobID(job.ID)
		log.Info().
			Int("attempts", job.Attempts).
			Int("max_attempts", job.MaxAttempts).
			Interface("retry_after", job.RetryAfter).
			Msg("Job failed, will retry")
	} else {
		// Job has exceeded max retries - mark as permanently failed
		job.Status = interfaces.StatusPermanentFailed
		job.RetryAfter = nil

		metrics.JobsFailedTotal.Inc()
		log := logger.WithJobID(job.ID)
		log.Info().Int("attempts", job.Attempts).Msg("Job permanently failed")
	}

	if err := m.store.UpdateJob(job); err != nil {
		return fmt.Errorf("failed to update failed job: %w", err)
	}

	return nil
}

// DeleteJob removes a job from the database
func (m *Manager) DeleteJob(id string) error {
	return m.store.DeleteJob(id)
}
