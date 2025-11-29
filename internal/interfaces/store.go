package interfaces

import (
	"fmt"
	"time"
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	StatusPending         JobStatus = "pending"
	StatusProcessing      JobStatus = "processing"
	StatusCompleted       JobStatus = "completed"
	StatusFailed          JobStatus = "failed"
	StatusRetrying        JobStatus = "retrying"
	StatusPermanentFailed JobStatus = "permanent_failed"
)

// Job represents a job in the queue
type Job struct {
	ID          string     `json:"id"`
	Type        string     `json:"type"`
	Payload     string     `json:"payload"`
	Status      JobStatus  `json:"status"`
	Result      string     `json:"result,omitempty"`
	Error       string     `json:"error,omitempty"`
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"max_attempts"`
	RetryAfter  *time.Time `json:"retry_after,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// String returns a string representation of the job
func (j *Job) String() string {
	return fmt.Sprintf("Job{ID: %s, Type: %s, Status: %s, Attempts: %d/%d}",
		j.ID, j.Type, j.Status, j.Attempts, j.MaxAttempts)
}

// CanRetry returns true if the job can be retried
func (j *Job) CanRetry() bool {
	return j.Attempts < j.MaxAttempts
}

// IncrementAttempts increments the attempt counter
func (j *Job) IncrementAttempts() {
	j.Attempts++
}

// SetRetryAfter sets the retry after time with exponential backoff
func (j *Job) SetRetryAfter(baseDelaySec int) {
	if baseDelaySec <= 0 {
		baseDelaySec = 1
	}

	// Exponential backoff: 2^(attempts-1) * baseDelay
	// For attempts 1,2,3: delays are 1s, 2s, 4s (if baseDelay=1)
	backoffDelay := time.Duration(1<<(j.Attempts-1)) * time.Duration(baseDelaySec) * time.Second

	// Cap the maximum delay at 5 minutes
	maxDelay := 5 * time.Minute
	if backoffDelay > maxDelay {
		backoffDelay = maxDelay
	}

	retryTime := time.Now().Add(backoffDelay)
	j.RetryAfter = &retryTime
}

// IsReadyForRetry returns true if the job is ready to be retried
func (j *Job) IsReadyForRetry() bool {
	if j.Status != StatusRetrying || j.RetryAfter == nil {
		return false
	}
	return time.Now().After(*j.RetryAfter)
}

// JobStore interface defines the database operations needed by the manager
type JobStore interface {
	CreateJob(job *Job) error
	GetJob(id string) (*Job, error)
	UpdateJob(job *Job) error
	GetPendingJob() (*Job, error)
	GetAllJobs() ([]*Job, error)
	DeleteJob(id string) error
}
