package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"egobot/internal/ai"
	"egobot/internal/config"
	"egobot/internal/processor"

	"github.com/gin-gonic/gin"
	"github.com/robfig/cron/v3"
	"go.uber.org/fx"
)

func NewRouter() *gin.Engine {
	r := gin.Default()

	// Health check endpoint for Railway
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message":   "pong",
			"status":    "healthy",
			"timestamp": time.Now().Format("2006-01-02 15:04:05"),
		})
	})

	// Cron status endpoint
	r.GET("/cron/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":      "egobot",
			"status":       "running",
			"cron_enabled": true,
			"next_run":     "6:00 AM CET daily",
			"timestamp":    time.Now().Format("2006-01-02 15:04:05"),
		})
	})

	// Root endpoint
	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "egobot - PDF Analysis Service",
			"version": "1.0.0",
			"endpoints": []string{
				"GET /ping - Health check",
				"GET /cron/status - Cron job status",
				"POST /extract - Extract entities from PDF",
			},
		})
	})
	r.POST("/extract", func(c *gin.Context) {
		// Parse multipart form
		err := c.Request.ParseMultipartForm(32 << 20) // 32MB max memory
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse multipart form: " + err.Error()})
			return
		}
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing PDF file: " + err.Error()})
			return
		}
		defer file.Close()

		entitiesStr := c.Request.FormValue("entities")
		if entitiesStr == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing entities field (should be a JSON array)"})
			return
		}
		var entities []string
		if err := json.Unmarshal([]byte(entitiesStr), &entities); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid entities JSON: " + err.Error()})
			return
		}

		// Pass the file (as multipart.File) and filename to the AI extractor
		result, err := ai.ExtractEntitiesFromPDFFile(context.Background(), file, header.Filename, entities)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, result)
	})
	return r
}

func RunServer(lc fx.Lifecycle, router *gin.Engine) {
	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	// Load configuration for cron job
	cfg, err := config.Load()
	if err != nil {
		panic("Failed to load configuration: " + err.Error())
	}

	// Create processor for cron job
	proc := processor.NewProcessor(cfg)

	// Set up cron scheduler
	scheduler := cron.New()

	// Clean up any existing cron entries (in case of restart)
	entries := scheduler.Entries()
	log.Printf("Removing %d existing cron entries to ensure only one job is running", len(entries))
	for _, entry := range entries {
		scheduler.Remove(entry.ID)
		log.Printf("🧹 Removed existing cron entry with ID %d", entry.ID)
	}

	// Use the schedule from config, or default to hourly for testing
	cronSchedule := cfg.ScheduleCron
	log.Printf("Using cron schedule found in config: %s", cronSchedule)
	log.Printf("🚀 Starting egobot service with internal cron")
	log.Printf("📅 Cron schedule: %s", cronSchedule)

	entryID, err := scheduler.AddFunc(cronSchedule, func() {
		log.Printf("🕕 Cron job triggered - running daily email processing")
		startTime := time.Now()

		if err := proc.ProcessWithRetry(); err != nil {
			log.Printf("❌ Cron job failed after %v: %v", time.Since(startTime), err)
		} else {
			log.Printf("✅ Cron job completed successfully in %v", time.Since(startTime))
		}
	})

	if err != nil {
		log.Printf("❌ Failed to add cron job: %v", err)
	} else {
		log.Printf("✅ Cron job scheduled with ID %d: %s", entryID, cronSchedule)
	}

	// Start the cron scheduler
	scheduler.Start()
	log.Printf("🌐 HTTP server starting on port 8080")

	lc.Append(fx.Hook{
		OnStart: func(context.Context) error {
			go func() {
				if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
					log.Printf("❌ Server error: %v", err)
				}
			}()
			return nil
		},
		OnStop: func(ctx context.Context) error {
			// Stop the cron scheduler
			scheduler.Stop()

			// Shutdown the server gracefully
			return server.Shutdown(ctx)
		},
	})
}

func main() {
	app := fx.New(
		fx.Provide(NewRouter),
		fx.Invoke(RunServer),
	)
	app.Run()
}
