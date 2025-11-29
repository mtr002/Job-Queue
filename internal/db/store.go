package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/mtr002/Job-Queue/internal/interfaces"
)

// Store handles database operations for jobs
type Store struct {
	db *sql.DB
}

// NewStore creates a new database store
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// CreateJob inserts a new job into the database
func (s *Store) CreateJob(job *interfaces.Job) error {
	query := `
		INSERT INTO jobs (id, type, payload, status, result, error, attempts, max_attempts, retry_after, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := s.db.Exec(query,
		job.ID, job.Type, job.Payload, job.Status, job.Result, job.Error,
		job.Attempts, job.MaxAttempts, job.RetryAfter, job.CreatedAt, job.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create job: %w", err)
	}

	return nil
}

// GetJob retrieves a job by ID
func (s *Store) GetJob(id string) (*interfaces.Job, error) {
	query := `
		SELECT id, type, payload, status, result, error, attempts, max_attempts, retry_after, created_at, updated_at
		FROM jobs WHERE id = $1
	`

	job := &interfaces.Job{}
	var retryAfter sql.NullTime

	err := s.db.QueryRow(query, id).Scan(
		&job.ID, &job.Type, &job.Payload, &job.Status, &job.Result, &job.Error,
		&job.Attempts, &job.MaxAttempts, &retryAfter, &job.CreatedAt, &job.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job with ID %s not found", id)
		}
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	if retryAfter.Valid {
		job.RetryAfter = &retryAfter.Time
	}

	return job, nil
}

// UpdateJob updates an existing job
func (s *Store) UpdateJob(job *interfaces.Job) error {
	query := `
		UPDATE jobs 
		SET status = $2, result = $3, error = $4, attempts = $5, retry_after = $6, updated_at = $7
		WHERE id = $1
	`

	job.UpdatedAt = time.Now()

	_, err := s.db.Exec(query,
		job.ID, job.Status, job.Result, job.Error, job.Attempts, job.RetryAfter, job.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update job: %w", err)
	}

	return nil
}

// GetPendingJob retrieves the next pending job for processing using SELECT FOR UPDATE to prevent race conditions
func (s *Store) GetPendingJob() (*interfaces.Job, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get the next pending job or a job that's ready for retry
	query := `
		SELECT id, type, payload, status, result, error, attempts, max_attempts, retry_after, created_at, updated_at
		FROM jobs 
		WHERE (status = 'pending') OR (status = 'retrying' AND retry_after <= NOW())
		ORDER BY created_at ASC
		LIMIT 1
		FOR UPDATE SKIP LOCKED
	`

	job := &interfaces.Job{}
	var retryAfter sql.NullTime

	err = tx.QueryRow(query).Scan(
		&job.ID, &job.Type, &job.Payload, &job.Status, &job.Result, &job.Error,
		&job.Attempts, &job.MaxAttempts, &retryAfter, &job.CreatedAt, &job.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No pending jobs
		}
		return nil, fmt.Errorf("failed to get pending job: %w", err)
	}

	if retryAfter.Valid {
		job.RetryAfter = &retryAfter.Time
	}

	// Mark as processing
	job.Status = interfaces.StatusProcessing
	job.UpdatedAt = time.Now()

	updateQuery := `UPDATE jobs SET status = $2, updated_at = $3 WHERE id = $1`
	_, err = tx.Exec(updateQuery, job.ID, job.Status, job.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to mark job as processing: %w", err)
	}

	if err = tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return job, nil
}

// GetAllJobs retrieves all jobs
func (s *Store) GetAllJobs() ([]*interfaces.Job, error) {
	query := `
		SELECT id, type, payload, status, result, error, attempts, max_attempts, retry_after, created_at, updated_at
		FROM jobs ORDER BY created_at DESC
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query jobs: %w", err)
	}
	defer rows.Close()

	var jobs []*interfaces.Job
	for rows.Next() {
		job := &interfaces.Job{}
		var retryAfter sql.NullTime

		err := rows.Scan(
			&job.ID, &job.Type, &job.Payload, &job.Status, &job.Result, &job.Error,
			&job.Attempts, &job.MaxAttempts, &retryAfter, &job.CreatedAt, &job.UpdatedAt)

		if err != nil {
			return nil, fmt.Errorf("failed to scan job: %w", err)
		}

		if retryAfter.Valid {
			job.RetryAfter = &retryAfter.Time
		}

		jobs = append(jobs, job)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return jobs, nil
}

// DeleteJob removes a job from the database
func (s *Store) DeleteJob(id string) error {
	query := `DELETE FROM jobs WHERE id = $1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("job with ID %s not found", id)
	}

	return nil
}
