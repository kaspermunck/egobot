package email

import (
	"testing"

	"github.com/emersion/go-imap"
)

func TestNewEmailFetcher(t *testing.T) {
	config := &Config{
		Server:   "imap.gmail.com",
		Port:     993,
		Username: "test@example.com",
		Password: "password123",
		Folder:   "INBOX",
	}

	fetcher := NewEmailFetcher(config)
	if fetcher == nil {
		t.Error("Expected fetcher to be created")
	}

	if fetcher.config != config {
		t.Error("Expected config to be set correctly")
	}
}

func TestEmailFetcher_ConfigValidation(t *testing.T) {
	// Test with valid config
	config := &Config{
		Server:   "imap.gmail.com",
		Port:     993,
		Username: "test@example.com",
		Password: "password123",
		Folder:   "INBOX",
	}

	fetcher := NewEmailFetcher(config)
	if fetcher.config.Server != "imap.gmail.com" {
		t.Errorf("Expected server to be imap.gmail.com, got %s", fetcher.config.Server)
	}

	if fetcher.config.Port != 993 {
		t.Errorf("Expected port to be 993, got %d", fetcher.config.Port)
	}
}

func TestEmailFetcher_FormatAddress(t *testing.T) {
	fetcher := NewEmailFetcher(&Config{})

	// Test with personal name
	addresses := []*imap.Address{
		{
			PersonalName: "John Doe",
			HostName:     "example.com",
			MailboxName:  "john",
		},
	}

	result := fetcher.formatAddress(addresses)
	expected := "John Doe <john@example.com>"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test without personal name
	addresses = []*imap.Address{
		{
			HostName:    "example.com",
			MailboxName: "john",
		},
	}

	result = fetcher.formatAddress(addresses)
	expected = "john@example.com"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}

	// Test empty addresses
	result = fetcher.formatAddress([]*imap.Address{})
	if result != "" {
		t.Errorf("Expected empty string, got %s", result)
	}
}
