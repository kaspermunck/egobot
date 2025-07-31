package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the application
type Config struct {
	// OpenAI settings
	OpenAIAPIKey string
	OpenAIStub   bool // If true, use stubbed responses instead of real API calls

	// Email settings
	IMAPServer   string
	IMAPPort     int
	IMAPUsername string
	IMAPPassword string
	IMAPFolder   string

	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	SMTPFrom     string
	SMTPTo       string

	// Processing settings
	EntitiesToTrack []string
	ScheduleCron    string
	MaxRetries      int
	RetryDelay      time.Duration
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	config := &Config{
		OpenAIAPIKey: getEnvOrDefault("OPENAI_API_KEY", ""),
		OpenAIStub:   getEnvBoolOrDefault("OPENAI_STUB", true), // Default to stubbed for safety

		IMAPServer:   getEnvOrDefault("IMAP_SERVER", "imap.gmail.com"),
		IMAPPort:     getEnvIntOrDefault("IMAP_PORT", 993),
		IMAPUsername: getEnvOrDefault("IMAP_USERNAME", ""),
		IMAPPassword: getEnvOrDefault("IMAP_PASSWORD", ""),
		IMAPFolder:   getEnvOrDefault("IMAP_FOLDER", "INBOX"),

		SMTPHost:     getEnvOrDefault("SMTP_HOST", "smtp.gmail.com"),
		SMTPPort:     getEnvIntOrDefault("SMTP_PORT", 587),
		SMTPUsername: getEnvOrDefault("SMTP_USERNAME", ""),
		SMTPPassword: getEnvOrDefault("SMTP_PASSWORD", ""),
		SMTPFrom:     getEnvOrDefault("SMTP_FROM", ""),
		SMTPTo:       getEnvOrDefault("SMTP_TO", ""),

		EntitiesToTrack: getEnvSliceOrDefault("ENTITIES_TO_TRACK", []string{"pikkemand"}),
		ScheduleCron:    getEnvOrDefault("SCHEDULE_CRON", "0 6 * * * *"), // Daily at 6 AM
		MaxRetries:      getEnvIntOrDefault("MAX_RETRIES", 3),
		RetryDelay:      getEnvDurationOrDefault("RETRY_DELAY", 5*time.Minute),
	}

	// Validate required fields
	if config.OpenAIAPIKey == "" && !config.OpenAIStub {
		return nil, fmt.Errorf("OPENAI_API_KEY is required when not using stubbed mode")
	}
	if config.IMAPUsername == "" {
		return nil, fmt.Errorf("IMAP_USERNAME is required")
	}
	if config.IMAPPassword == "" {
		return nil, fmt.Errorf("IMAP_PASSWORD is required")
	}
	if config.SMTPFrom == "" {
		return nil, fmt.Errorf("SMTP_FROM is required")
	}
	if config.SMTPTo == "" {
		return nil, fmt.Errorf("SMTP_TO is required")
	}

	return config, nil
}

// Helper functions for environment variables
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBoolOrDefault(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDurationOrDefault(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvSliceOrDefault(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Try to parse as JSON array first
		var jsonArray []string
		if err := json.Unmarshal([]byte(value), &jsonArray); err == nil {
			return jsonArray
		}

		// Fallback to comma-separated values
		if strings.Contains(value, ",") {
			return strings.Split(value, ",")
		}

		// If it's a single value, return as array with one item
		return []string{value}
	}
	return defaultValue
}
