package scheduler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/robfig/cron/v3"
)

// Scheduler manages scheduled jobs using cron
type Scheduler struct {
	cron      *cron.Cron
	processor Processor
	config    *Config
}

// Processor interface for the email processor
type Processor interface {
	ProcessWithRetry() error
}

// Config holds scheduler configuration
type Config struct {
	CronSchedule string
	MaxRetries   int
	RetryDelay   time.Duration
}

// NewScheduler creates a new scheduler
func NewScheduler(processor Processor, config *Config) *Scheduler {
	return &Scheduler{
		cron:      cron.New(cron.WithSeconds()),
		processor: processor,
		config:    config,
	}
}

// Start starts the scheduler and begins processing
func (s *Scheduler) Start() error {
	log.Printf("Starting scheduler with cron schedule: %s", s.config.CronSchedule)

	// Add the processing job
	entryID, err := s.cron.AddFunc(s.config.CronSchedule, s.runProcessingJob)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	log.Printf("Scheduled job with ID: %d", entryID)

	// Start the cron scheduler
	s.cron.Start()
	log.Printf("Scheduler started successfully")

	return nil
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	log.Printf("Stopping scheduler")
	s.cron.Stop()
	log.Printf("Scheduler stopped")
}

// runProcessingJob runs the email processing job
func (s *Scheduler) runProcessingJob() {
	log.Printf("Running scheduled email processing job at %s", time.Now().Format("2006-01-02 15:04:05"))

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	// Run processing in a goroutine to avoid blocking
	go func() {
		if err := s.processor.ProcessWithRetry(); err != nil {
			log.Printf("Scheduled processing job failed: %v", err)
		} else {
			log.Printf("Scheduled processing job completed successfully")
		}
	}()

	// Wait for completion or timeout
	select {
	case <-ctx.Done():
		log.Printf("Processing job timed out after 30 minutes")
	case <-time.After(29 * time.Minute): // Give 1 minute buffer
		log.Printf("Processing job completed within time limit")
	}
}

// RunOnce runs the processing job once immediately
func (s *Scheduler) RunOnce() error {
	log.Printf("Running one-time email processing job")
	return s.processor.ProcessWithRetry()
}

// GetNextRunTime returns the next scheduled run time
func (s *Scheduler) GetNextRunTime() (time.Time, error) {
	entries := s.cron.Entries()
	if len(entries) == 0 {
		return time.Time{}, fmt.Errorf("no scheduled jobs found")
	}
	return entries[0].Next, nil
}

// GetScheduleInfo returns information about the current schedule
func (s *Scheduler) GetScheduleInfo() map[string]interface{} {
	entries := s.cron.Entries()
	if len(entries) == 0 {
		return map[string]interface{}{
			"status": "no_jobs_scheduled",
		}
	}

	entry := entries[0]
	return map[string]interface{}{
		"schedule":   s.config.CronSchedule,
		"next_run":   entry.Next.Format("2006-01-02 15:04:05"),
		"last_run":   entry.Prev.Format("2006-01-02 15:04:05"),
		"is_running": !entry.Next.IsZero(),
		"job_count":  len(entries),
	}
}
