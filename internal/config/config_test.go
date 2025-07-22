package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadConfig(t *testing.T) {
	// Test with minimal required environment variables
	os.Setenv("IMAP_USERNAME", "test@example.com")
	os.Setenv("IMAP_PASSWORD", "password123")
	os.Setenv("SMTP_FROM", "from@example.com")
	os.Setenv("SMTP_TO", "to@example.com")
	os.Setenv("OPENAI_STUB", "true")

	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Test default values
	if config.IMAPServer != "imap.gmail.com" {
		t.Errorf("Expected IMAP server to be imap.gmail.com, got %s", config.IMAPServer)
	}
	if config.IMAPPort != 993 {
		t.Errorf("Expected IMAP port to be 993, got %d", config.IMAPPort)
	}
	if !config.OpenAIStub {
		t.Error("Expected OpenAIStub to be true by default")
	}
	if config.ScheduleCron != "0 9 * * *" {
		t.Errorf("Expected schedule cron to be '0 9 * * *', got %s", config.ScheduleCron)
	}
}

func TestLoadConfigValidation(t *testing.T) {
	// Test missing required fields
	os.Clearenv()
	os.Setenv("OPENAI_STUB", "true")

	_, err := Load()
	if err == nil {
		t.Error("Expected error when required fields are missing")
	}

	// Test with all required fields
	os.Setenv("IMAP_USERNAME", "test@example.com")
	os.Setenv("IMAP_PASSWORD", "password123")
	os.Setenv("SMTP_FROM", "from@example.com")
	os.Setenv("SMTP_TO", "to@example.com")

	config, err := Load()
	if err != nil {
		t.Fatalf("Failed to load config with all required fields: %v", err)
	}

	if config.IMAPUsername != "test@example.com" {
		t.Errorf("Expected IMAP username to be 'test@example.com', got %s", config.IMAPUsername)
	}
}

func TestEnvironmentVariableHelpers(t *testing.T) {
	// Test getEnvOrDefault
	os.Setenv("TEST_STRING", "test_value")
	if result := getEnvOrDefault("TEST_STRING", "default"); result != "test_value" {
		t.Errorf("Expected 'test_value', got %s", result)
	}
	if result := getEnvOrDefault("NONEXISTENT", "default"); result != "default" {
		t.Errorf("Expected 'default', got %s", result)
	}

	// Test getEnvIntOrDefault
	os.Setenv("TEST_INT", "42")
	if result := getEnvIntOrDefault("TEST_INT", 0); result != 42 {
		t.Errorf("Expected 42, got %d", result)
	}
	if result := getEnvIntOrDefault("NONEXISTENT", 10); result != 10 {
		t.Errorf("Expected 10, got %d", result)
	}

	// Test getEnvBoolOrDefault
	os.Setenv("TEST_BOOL", "true")
	if result := getEnvBoolOrDefault("TEST_BOOL", false); !result {
		t.Error("Expected true, got false")
	}
	if result := getEnvBoolOrDefault("NONEXISTENT", true); !result {
		t.Error("Expected true, got false")
	}

	// Test getEnvDurationOrDefault
	os.Setenv("TEST_DURATION", "5m")
	if result := getEnvDurationOrDefault("TEST_DURATION", time.Minute); result != 5*time.Minute {
		t.Errorf("Expected 5m, got %v", result)
	}
	if result := getEnvDurationOrDefault("NONEXISTENT", 10*time.Second); result != 10*time.Second {
		t.Errorf("Expected 10s, got %v", result)
	}
}
