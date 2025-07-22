package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"egobot/internal/config"
	"egobot/internal/processor"
	"egobot/internal/scheduler"
)

func main() {
	// Parse command line flags
	var (
		runOnce      = flag.Bool("once", false, "Run processing once and exit")
		showSchedule = flag.Bool("schedule", false, "Show current schedule information")
	)
	flag.Parse()

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create processor
	proc := processor.NewProcessor(cfg)

	// Create scheduler
	schedulerConfig := &scheduler.Config{
		CronSchedule: cfg.ScheduleCron,
		MaxRetries:   cfg.MaxRetries,
		RetryDelay:   cfg.RetryDelay,
	}
	sched := scheduler.NewScheduler(proc, schedulerConfig)

	// Handle different modes
	if *showSchedule {
		showScheduleInfo(sched)
		return
	}

	if *runOnce {
		runProcessingOnce(proc)
		return
	}

	// Start scheduled processing
	runScheduledProcessing(sched)
}

// showScheduleInfo displays current schedule information
func showScheduleInfo(sched *scheduler.Scheduler) {
	info := sched.GetScheduleInfo()

	fmt.Println("📅 Email Processor Schedule Information")
	fmt.Println("=====================================")

	if status, ok := info["status"].(string); ok && status == "no_jobs_scheduled" {
		fmt.Println("❌ No jobs currently scheduled")
		return
	}

	if schedule, ok := info["schedule"].(string); ok {
		fmt.Printf("📋 Schedule: %s\n", schedule)
	}

	if nextRun, ok := info["next_run"].(string); ok {
		fmt.Printf("⏰ Next Run: %s\n", nextRun)
	}

	if lastRun, ok := info["last_run"].(string); ok {
		fmt.Printf("🕐 Last Run: %s\n", lastRun)
	}

	if isRunning, ok := info["is_running"].(bool); ok {
		if isRunning {
			fmt.Println("🟢 Status: Active")
		} else {
			fmt.Println("🔴 Status: Inactive")
		}
	}

	if jobCount, ok := info["job_count"].(int); ok {
		fmt.Printf("📊 Jobs: %d\n", jobCount)
	}
}

// runProcessingOnce runs the processing job once
func runProcessingOnce(proc *processor.Processor) {
	fmt.Println("🚀 Running one-time email processing...")

	start := time.Now()
	if err := proc.ProcessWithRetry(); err != nil {
		log.Fatalf("Processing failed: %v", err)
	}

	duration := time.Since(start)
	fmt.Printf("✅ Processing completed successfully in %v\n", duration)
}

// runScheduledProcessing starts the scheduler and runs continuously
func runScheduledProcessing(sched *scheduler.Scheduler) {
	fmt.Println("🔄 Starting scheduled email processor...")

	// Start the scheduler
	if err := sched.Start(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	// Show initial schedule info
	showScheduleInfo(sched)

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("\n📧 Email processor is running...")
	fmt.Println("Press Ctrl+C to stop")

	// Wait for shutdown signal
	<-sigChan
	fmt.Println("\n🛑 Shutting down email processor...")

	// Stop the scheduler
	sched.Stop()

	fmt.Println("✅ Email processor stopped gracefully")
}

// gracefulShutdown handles graceful shutdown of the application
func gracefulShutdown(ctx context.Context, sched *scheduler.Scheduler) {
	// Create a channel for shutdown completion
	done := make(chan bool)

	go func() {
		fmt.Println("🛑 Stopping scheduler...")
		sched.Stop()
		done <- true
	}()

	// Wait for shutdown or timeout
	select {
	case <-done:
		fmt.Println("✅ Shutdown completed")
	case <-ctx.Done():
		fmt.Println("⏰ Shutdown timed out")
	}
}
