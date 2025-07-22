package email

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// EmailMessage represents a processed email with attachments
type EmailMessage struct {
	ID          string
	Subject     string
	From        string
	Date        time.Time
	Attachments []Attachment
}

// Attachment represents a file attachment from an email
type Attachment struct {
	Filename    string
	ContentType string
	Data        io.Reader
}

// EmailFetcher handles IMAP email fetching
type EmailFetcher struct {
	config *Config
}

// Config holds email fetching configuration
type Config struct {
	Server   string
	Port     int
	Username string
	Password string
	Folder   string
}

// NewEmailFetcher creates a new email fetcher
func NewEmailFetcher(config *Config) *EmailFetcher {
	return &EmailFetcher{
		config: config,
	}
}

// FetchPDFEmails fetches emails with PDF attachments from the last 24 hours
func (f *EmailFetcher) FetchPDFEmails() ([]EmailMessage, error) {
	log.Printf("Connecting to IMAP server: %s:%d", f.config.Server, f.config.Port)

	// Connect to IMAP server
	c, err := client.DialTLS(fmt.Sprintf("%s:%d", f.config.Server, f.config.Port), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to IMAP server: %w", err)
	}
	defer c.Logout()

	// Login
	if err := c.Login(f.config.Username, f.config.Password); err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	// Select mailbox
	_, err = c.Select(f.config.Folder, false)
	if err != nil {
		return nil, fmt.Errorf("failed to select mailbox: %w", err)
	}

	// Search for emails from the last 24 hours
	since := time.Now().Add(-24 * time.Hour)
	criteria := imap.NewSearchCriteria()
	criteria.Since = since

	uids, err := c.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}

	if len(uids) == 0 {
		log.Printf("No emails found in the last 24 hours")
		return []EmailMessage{}, nil
	}

	log.Printf("Found %d emails in the last 24 hours", len(uids))

	// For now, return empty results since we need to implement proper attachment parsing
	// This is a placeholder that will be enhanced in the next phase
	log.Printf("Email fetching implemented - attachment parsing coming in next phase")
	return []EmailMessage{}, nil
}

// formatAddress formats an email address
func (f *EmailFetcher) formatAddress(addresses []*imap.Address) string {
	if len(addresses) == 0 {
		return ""
	}
	addr := addresses[0]
	if addr.PersonalName != "" {
		return fmt.Sprintf("%s <%s>", addr.PersonalName, addr.Address())
	}
	return addr.Address()
}
