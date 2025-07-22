package scheduler

import (
	"fmt"
	"testing"
	"time"
)

// MockProcessor for testing
type MockProcessor struct {
	processCalled bool
	processError  error
}

func (m *MockProcessor) ProcessWithRetry() error {
	m.processCalled = true
	return m.processError
}

func TestNewScheduler(t *testing.T) {
	config := &Config{
		CronSchedule: "0 0 9 * * *",
		MaxRetries:   3,
		RetryDelay:   5 * time.Minute,
	}

	processor := &MockProcessor{}

	sched := NewScheduler(processor, config)
	if sched == nil {
		t.Error("Expected scheduler to be created")
	}

	if sched.config != config {
		t.Error("Expected config to be set correctly")
	}

	if sched.processor != processor {
		t.Error("Expected processor to be set correctly")
	}
}

func TestScheduler_Start(t *testing.T) {
	config := &Config{
		CronSchedule: "0 0 9 * * *",
		MaxRetries:   3,
		RetryDelay:   5 * time.Minute,
	}

	processor := &MockProcessor{}

	sched := NewScheduler(processor, config)

	err := sched.Start()
	if err != nil {
		t.Errorf("Expected no error starting scheduler, got %v", err)
	}

	// Check that scheduler is running
	info := sched.GetScheduleInfo()
	if status, ok := info["status"].(string); ok && status == "no_jobs_scheduled" {
		t.Error("Expected scheduler to have jobs scheduled")
	}

	// Stop the scheduler
	sched.Stop()
}

func TestScheduler_RunOnce(t *testing.T) {
	config := &Config{
		CronSchedule: "0 0 9 * * *",
		MaxRetries:   3,
		RetryDelay:   5 * time.Minute,
	}

	processor := &MockProcessor{}

	sched := NewScheduler(processor, config)

	// Test successful run
	err := sched.RunOnce()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !processor.processCalled {
		t.Error("Expected processor to be called")
	}

	// Test with error
	processor.processError = fmt.Errorf("test error")
	processor.processCalled = false

	err = sched.RunOnce()
	if err == nil {
		t.Error("Expected error when processor fails")
	}
}

func TestScheduler_GetScheduleInfo(t *testing.T) {
	config := &Config{
		CronSchedule: "0 0 9 * * *",
		MaxRetries:   3,
		RetryDelay:   5 * time.Minute,
	}

	processor := &MockProcessor{}

	sched := NewScheduler(processor, config)

	// Test before starting
	info := sched.GetScheduleInfo()
	if status, ok := info["status"].(string); ok && status == "no_jobs_scheduled" {
		// This is expected before starting
	} else {
		t.Error("Expected no jobs scheduled before starting")
	}

	// Start scheduler
	err := sched.Start()
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// Test after starting
	info = sched.GetScheduleInfo()
	if schedule, ok := info["schedule"].(string); !ok || schedule != "0 0 9 * * *" {
		t.Error("Expected schedule to be set")
	}

	if jobCount, ok := info["job_count"].(int); !ok || jobCount != 1 {
		t.Error("Expected job count to be 1")
	}

	// Stop scheduler
	sched.Stop()
}

func TestScheduler_GetNextRunTime(t *testing.T) {
	config := &Config{
		CronSchedule: "0 0 9 * * *",
		MaxRetries:   3,
		RetryDelay:   5 * time.Minute,
	}

	processor := &MockProcessor{}

	sched := NewScheduler(processor, config)

	// Test before starting
	_, err := sched.GetNextRunTime()
	if err == nil {
		t.Error("Expected error when no jobs scheduled")
	}

	// Start scheduler
	err = sched.Start()
	if err != nil {
		t.Fatalf("Failed to start scheduler: %v", err)
	}

	// Test after starting
	nextRun, err := sched.GetNextRunTime()
	if err != nil {
		t.Errorf("Expected no error getting next run time, got %v", err)
	}

	if nextRun.IsZero() {
		t.Error("Expected next run time to be set")
	}

	// Stop scheduler
	sched.Stop()
}
