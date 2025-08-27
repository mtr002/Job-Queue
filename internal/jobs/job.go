package jobs

import (
	"fmt"
	"time"
)

// JobStatus represents the current state of a job
type JobStatus string

const (
	StatusPending    JobStatus = "pending"
	StatusProcessing JobStatus = "processing"
	StatusCompleted  JobStatus = "completed"
	StatusFailed     JobStatus = "failed"
)

// Job represents a job in the queue
type Job struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Payload   string    `json:"payload"`
	Status    JobStatus `json:"status"`
	Result    string    `json:"result,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Error     string    `json:"error,omitempty"`
}

// String returns a string representation of the job
func (j *Job) String() string {
	return fmt.Sprintf("Job{ID: %s, Type: %s, Status: %s}", j.ID, j.Type, j.Status)
}
