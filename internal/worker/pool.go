package worker

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"sync"
	"time"

	"github.com/mtr002/Job-Queue/internal/interfaces"
	"github.com/mtr002/Job-Queue/internal/jobs"
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
	log.Printf("Starting worker pool with %d workers", p.workerCount)

	for i := 0; i < p.workerCount; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}

// Stop gracefully shuts down the worker pool
func (p *Pool) Stop() {
	log.Println("Stopping worker pool...")
	p.cancel()
	p.wg.Wait()
	log.Println("Worker pool stopped")
}

// worker is the main worker goroutine that polls database for jobs
func (p *Pool) worker(id int) {
	defer p.wg.Done()

	log.Printf("Worker %d started", id)

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-p.ctx.Done():
			log.Printf("Worker %d shutting down", id)
			return
		case <-ticker.C:
			// Poll for a pending job
			job, err := p.manager.GetPendingJob()
			if err != nil {
				log.Printf("Worker %d: error getting pending job: %v", id, err)
				continue
			}

			if job != nil {
				p.processJob(id, job)
			}
			// If no job is available, continue polling
		}
	}
}

// processJob handles the processing of a single job with retry logic
func (p *Pool) processJob(workerID int, job *interfaces.Job) {
	log.Printf("Worker %d processing job %s (type: %s, attempt: %d/%d)",
		workerID, job.ID, job.Type, job.Attempts+1, job.MaxAttempts)

	// Process the job
	result, err := p.processor.Process(job)
	if err != nil {
		log.Printf("Worker %d: job %s failed: %v", workerID, job.ID, err)
		if updateErr := p.manager.UpdateJobFailed(job, err.Error()); updateErr != nil {
			log.Printf("Worker %d: failed to update failed job %s: %v", workerID, job.ID, updateErr)
		}
		return
	}

	// Mark job as completed
	if err := p.manager.UpdateJobCompleted(job, result); err != nil {
		log.Printf("Worker %d: failed to update job %s to completed: %v", workerID, job.ID, err)
		return
	}

	log.Printf("Worker %d completed job %s", workerID, job.ID)
}

// DefaultJobProcessor provides a simple implementation for demonstration
type DefaultJobProcessor struct{}

// Process implements JobProcessor interface with some mock job processing
func (d *DefaultJobProcessor) Process(job *interfaces.Job) (string, error) {
	switch job.Type {
	case "echo":
		// Simple echo job - just return the payload
		return fmt.Sprintf("Echo: %s", job.Payload), nil

	case "uppercase":
		// Convert payload to uppercase
		return fmt.Sprintf("UPPERCASE: %s", job.Payload), nil

	case "slow":
		// Simulate slow processing
		// Generate random number between 1-5 for sleep duration
		n, err := rand.Int(rand.Reader, big.NewInt(5))
		if err != nil {
			n = big.NewInt(2) // fallback to 2 seconds
		}
		sleepDuration := time.Duration(n.Int64()+1) * time.Second
		log.Printf("Job %s sleeping for %v", job.ID, sleepDuration)
		time.Sleep(sleepDuration)
		return fmt.Sprintf("Slow job completed after %v", sleepDuration), nil

	case "fail":
		// Simulate a failing job
		return "", fmt.Errorf("simulated job failure")

	default:
		return "", fmt.Errorf("unknown job type: %s", job.Type)
	}
}
