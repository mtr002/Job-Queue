package worker

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/mtr002/Job-Queue/internal/interfaces"
	"github.com/mtr002/Job-Queue/internal/jobs"
	"github.com/mtr002/Job-Queue/internal/logger"
	"github.com/mtr002/Job-Queue/internal/metrics"
)

// JobProcessor defines the interface for processing different job types
type JobProcessor interface {
	Process(job *interfaces.Job) (string, error)
}

// Pool represents a worker pool that processes jobs from database
type Pool struct {
	manager      *jobs.Manager
	processor    JobProcessor
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	workerCount  int
	pollInterval time.Duration // How often to poll for new jobs
}

// NewPool creates a new worker pool with database polling
func NewPool(manager *jobs.Manager, processor JobProcessor, workerCount int) *Pool {
	ctx, cancel := context.WithCancel(context.Background())
	return &Pool{
		manager:      manager,
		processor:    processor,
		workerCount:  workerCount,
		ctx:          ctx,
		cancel:       cancel,
		pollInterval: 1 * time.Second, // Poll every second
	}
}

// Start begins processing jobs with the specified number of workers
func (p *Pool) Start() {
	logger.Logger.Info().Int("worker_count", p.workerCount).Msg("Starting worker pool")
	metrics.ActiveWorkers.Set(float64(p.workerCount))

	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop gracefully shuts down the worker pool
func (p *Pool) Stop() {
	logger.Logger.Info().Msg("Stopping worker pool")
	p.cancel()
	p.wg.Wait()
	metrics.ActiveWorkers.Set(0)
	logger.Logger.Info().Msg("Worker pool stopped")
}

// worker is the main worker goroutine that polls database for jobs
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	logger.Logger.Info().Int("worker_id", id).Msg("Worker started")

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			logger.Logger.Info().Int("worker_id", id).Msg("Worker shutting down")
			return
		case <-ticker.C:
			job, err := p.manager.GetPendingJob()
			if err != nil {
				logger.Logger.Error().Int("worker_id", id).Err(err).Msg("Error getting pending job")
				continue
			}

			if job != nil {
				p.processJob(id, job)
			}
		}
	}
}

// processJob handles the processing of a single job with retry logic
func (p *Pool) processJob(workerID int, job *interfaces.Job) {
	startTime := time.Now()
	logger.Logger.Info().
		Int("worker_id", workerID).
		Str("job_id", job.ID).
		Str("type", job.Type).
		Int("attempt", job.Attempts+1).
		Int("max_attempts", job.MaxAttempts).
		Msg("Processing job")

	result, err := p.processor.Process(job)
	duration := time.Since(startTime).Seconds()
	metrics.JobProcessingDuration.Observe(duration)

	if err != nil {
		logger.Logger.Error().
			Int("worker_id", workerID).
			Str("job_id", job.ID).
			Err(err).
			Msg("Job processing failed")
		if updateErr := p.manager.UpdateJobFailed(job, err.Error()); updateErr != nil {
			logger.Logger.Error().
				Int("worker_id", workerID).
				Str("job_id", job.ID).
				Err(updateErr).
				Msg("Failed to update failed job")
		}
		return
	}

	if err := p.manager.UpdateJobCompleted(job, result); err != nil {
		logger.Logger.Error().
			Int("worker_id", workerID).
			Str("job_id", job.ID).
			Err(err).
			Msg("Failed to update job as completed")
		return
	}

	logger.Logger.Info().
		Int("worker_id", workerID).
		Str("job_id", job.ID).
		Msg("Job completed")
}

// DefaultJobProcessor provides a simple implementation
type DefaultJobProcessor struct{}

// Process implements JobProcessor interface
func (d *DefaultJobProcessor) Process(job *interfaces.Job) (string, error) {
	switch job.Type {
	case "echo":
		return fmt.Sprintf("Echo: %s", job.Payload), nil

	case "uppercase":
		return fmt.Sprintf("UPPERCASE: %s", job.Payload), nil

	case "slow":
		n, err := rand.Int(rand.Reader, big.NewInt(5))
		if err != nil {
			n = big.NewInt(2)
		}
		sleepDuration := time.Duration(n.Int64()+1) * time.Second
		logger.Logger.Debug().Str("job_id", job.ID).Dur("duration", sleepDuration).Msg("Slow job sleeping")
		time.Sleep(sleepDuration)
		return fmt.Sprintf("Slow job completed after %v", sleepDuration), nil

	case "fail":
		return "", fmt.Errorf("simulated job failure")

	default:
		return "", fmt.Errorf("unknown job type: %s", job.Type)
	}
}
