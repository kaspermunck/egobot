package email

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/mail"
	"strings"
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

	log.Printf("Searching for emails since: %s", since.Format("2006-01-02 15:04:05"))
	uids, err := c.Search(criteria)
	if err != nil {
		return nil, fmt.Errorf("failed to search emails: %w", err)
	}

	if len(uids) == 0 {
		log.Printf("No emails found in the last 24 hours")
		return []EmailMessage{}, nil
	}

	log.Printf("Found %d emails in the last 24 hours", len(uids))

	// Fetch messages with full content
	seqset := new(imap.SeqSet)
	seqset.AddNum(uids...)

	messages := make(chan *imap.Message, 10)
	done := make(chan error, 1)

	// Fetch full message content including body
	log.Printf("Fetching %d messages with full content", len(uids))
	go func() {
		done <- c.Fetch(seqset, []imap.FetchItem{imap.FetchEnvelope, imap.FetchRFC822, imap.FetchUid, imap.FetchFlags}, messages)
	}()

	var emailMessages []EmailMessage
	for msg := range messages {
		log.Printf("Processing message UID: %d, Subject: %s", msg.Uid, msg.Envelope.Subject)
		emailMsg, err := f.processMessage(msg)
		if err != nil {
			log.Printf("Error processing message: %v", err)
			continue
		}
		if len(emailMsg.Attachments) > 0 {
			emailMessages = append(emailMessages, emailMsg)
		}
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	log.Printf("Successfully processed %d emails with PDF attachments", len(emailMessages))
	return emailMessages, nil
}

// processMessage processes a single email message
func (f *EmailFetcher) processMessage(msg *imap.Message) (EmailMessage, error) {
	emailMsg := EmailMessage{
		ID:          fmt.Sprintf("%d", msg.Uid),
		Subject:     msg.Envelope.Subject,
		From:        f.formatAddress(msg.Envelope.From),
		Date:        msg.Envelope.Date,
		Attachments: []Attachment{},
	}

	// Process message body to find attachments
	if err := f.processMessageBody(msg, &emailMsg); err != nil {
		return emailMsg, fmt.Errorf("failed to process message body: %w", err)
	}

	return emailMsg, nil
}

// processMessageBody processes the body of an email message
func (f *EmailFetcher) processMessageBody(msg *imap.Message, emailMsg *EmailMessage) error {
	// Try to get the message body using different approaches
	var messageBody io.Reader

	// Method 1: Try RFC822 (full message) - this should be available since we fetched it
	messageBody = msg.GetBody(&imap.BodySectionName{})
	if messageBody == nil {
		return fmt.Errorf("failed to get message body - RFC822 content not available")
	}

	// Parse the message using MIME
	entity, err := mail.ReadMessage(messageBody)
	if err != nil {
		return fmt.Errorf("failed to read message: %w", err)
	}

	return f.processEntity(entity, emailMsg)
}

// processEntity recursively processes email entities (multipart messages)
func (f *EmailFetcher) processEntity(entity *mail.Message, emailMsg *EmailMessage) error {
	// Check if this is a multipart message
	mediaType, params, err := mime.ParseMediaType(entity.Header.Get("Content-Type"))
	if err != nil {
		mediaType = "text/plain"
	}

	if strings.HasPrefix(mediaType, "multipart/") {
		// Handle multipart messages
		boundary := params["boundary"]
		if boundary == "" {
			return fmt.Errorf("multipart message without boundary")
		}

		mr := multipart.NewReader(entity.Body, boundary)
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return fmt.Errorf("failed to read multipart: %w", err)
			}

			// Recursively process each part
			if err := f.processPart(part, emailMsg); err != nil {
				log.Printf("Error processing part: %v", err)
				continue
			}
		}
	} else {
		// Handle single part message
		return f.processSinglePart(entity, emailMsg)
	}

	return nil
}

// processSinglePart processes a single-part message
func (f *EmailFetcher) processSinglePart(entity *mail.Message, emailMsg *EmailMessage) error {
	// Check if this is a PDF attachment
	contentType := entity.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/pdf") {
		filename := entity.Header.Get("Content-Disposition")
		if filename == "" {
			filename = "attachment.pdf"
		}

		// Clean up filename
		filename = strings.Trim(filename, `"`)
		if strings.HasPrefix(filename, "filename=") {
			filename = strings.TrimPrefix(filename, "filename=")
			filename = strings.Trim(filename, `"`)
		}

		// Read the attachment data
		data, err := io.ReadAll(entity.Body)
		if err != nil {
			return fmt.Errorf("failed to read attachment: %w", err)
		}

		attachment := Attachment{
			Filename:    filename,
			ContentType: contentType,
			Data:        bytes.NewReader(data),
		}
		emailMsg.Attachments = append(emailMsg.Attachments, attachment)
		log.Printf("Found PDF attachment: %s", filename)
	}

	return nil
}

// processPart processes a single message part
func (f *EmailFetcher) processPart(part *multipart.Part, emailMsg *EmailMessage) error {
	// Check if this is a PDF attachment
	contentType := part.Header.Get("Content-Type")
	if strings.Contains(contentType, "application/pdf") {
		filename := part.FileName()
		if filename == "" {
			filename = "attachment.pdf"
		}

		// Read the attachment data
		data, err := io.ReadAll(part)
		if err != nil {
			return fmt.Errorf("failed to read attachment: %w", err)
		}

		attachment := Attachment{
			Filename:    filename,
			ContentType: contentType,
			Data:        bytes.NewReader(data),
		}
		emailMsg.Attachments = append(emailMsg.Attachments, attachment)
		log.Printf("Found PDF attachment: %s", filename)
	}

	return nil
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
