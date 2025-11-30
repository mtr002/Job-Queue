package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	JobsSubmittedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "jobqueue_jobs_submitted_total",
		Help: "Total number of jobs submitted",
	})

	JobsCompletedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "jobqueue_jobs_completed_total",
		Help: "Total number of jobs completed successfully",
	})

	JobsFailedTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "jobqueue_jobs_failed_total",
		Help: "Total number of jobs that failed permanently",
	})

	JobProcessingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "jobqueue_job_processing_duration_seconds",
		Help:    "Time taken to process jobs in seconds",
		Buckets: prometheus.DefBuckets,
	})

	ActiveWorkers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "jobqueue_active_workers",
		Help: "Current number of active workers",
	})

	PendingJobs = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "jobqueue_pending_jobs",
		Help: "Current number of pending jobs",
	})
)
