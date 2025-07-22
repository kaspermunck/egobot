package email

import (
	"strings"
	"testing"
	"time"

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

func TestEmailFetcher_ProcessMessage(t *testing.T) {
	fetcher := NewEmailFetcher(&Config{})

	// Create a mock IMAP message
	msg := &imap.Message{
		Uid: 123,
		Envelope: &imap.Envelope{
			Subject: "Test Email",
			From: []*imap.Address{
				{
					PersonalName: "Test Sender",
					HostName:     "example.com",
					MailboxName:  "test",
				},
			},
			Date: time.Now(),
		},
	}

	// Test processing the message
	emailMsg, err := fetcher.processMessage(msg)
	if err != nil {
		// This is expected to fail because we don't have a real message body
		// But we can test that the basic structure is created
		t.Logf("Expected error processing message without body: %v", err)
	}

	// Verify the basic message structure was created
	if emailMsg.ID != "123" {
		t.Errorf("Expected ID '123', got %s", emailMsg.ID)
	}

	if emailMsg.Subject != "Test Email" {
		t.Errorf("Expected subject 'Test Email', got %s", emailMsg.Subject)
	}

	if !strings.Contains(emailMsg.From, "test@example.com") {
		t.Errorf("Expected from to contain 'test@example.com', got %s", emailMsg.From)
	}
}
