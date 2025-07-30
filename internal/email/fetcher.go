package email

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"net/http"
	"net/mail"
	"regexp"
	"strings"
	"time"

	"github.com/emersion/go-imap"
	"github.com/emersion/go-imap/client"
)

// EmailMessage represents a processed email with attachments
type EmailMessage struct {
	ID             string
	Subject        string
	From           string
	Date           time.Time
	Attachments    []Attachment
	PDFURLs        []string        // PDF URLs found in the email
	processedLinks map[string]bool // Track processed PDF links to avoid duplicates
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

// FetchPDFEmails fetches emails with PDF links from the last 24 hours
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
		if len(emailMsg.PDFURLs) > 0 {
			emailMessages = append(emailMessages, emailMsg)
		}
	}

	if err := <-done; err != nil {
		return nil, fmt.Errorf("failed to fetch messages: %w", err)
	}

	log.Printf("Successfully processed %d emails with PDF URLs", len(emailMessages))
	return emailMessages, nil
}

// processMessage processes a single email message
func (f *EmailFetcher) processMessage(msg *imap.Message) (EmailMessage, error) {
	emailMsg := EmailMessage{
		ID:             fmt.Sprintf("%d", msg.Uid),
		Subject:        msg.Envelope.Subject,
		From:           f.formatAddress(msg.Envelope.From),
		Date:           msg.Envelope.Date,
		Attachments:    []Attachment{},
		PDFURLs:        []string{},            // Initialize PDF URLs slice
		processedLinks: make(map[string]bool), // Initialize the processed links map
	}

	// Check if this is a Statstidende email with PDF link
	if f.isStatstidendeEmail(msg.Envelope.Subject) {
		log.Printf("Found Statstidende email: %s", msg.Envelope.Subject)

		// Process message body to find PDF links
		if err := f.processMessageBody(msg, &emailMsg); err != nil {
			return emailMsg, fmt.Errorf("failed to process message body: %w", err)
		}
	}

	return emailMsg, nil
}

// isStatstidendeEmail checks if the email is from Statstidende with PDF content
func (f *EmailFetcher) isStatstidendeEmail(subject string) bool {
	// Check for Statstidende emails with PDF content
	statstidendePatterns := []string{
		"Dagens kundgørelse",
		"Statstidende",
		"PDF",
	}

	subjectLower := strings.ToLower(subject)
	for _, pattern := range statstidendePatterns {
		if strings.Contains(strings.ToLower(subjectLower), strings.ToLower(pattern)) {
			return true
		}
	}

	return false
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
	// Check if this is a PDF attachment (legacy support)
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
	} else {
		// Look for PDF links in text content
		return f.extractPDFLinks(entity, emailMsg)
	}

	return nil
}

// processPart processes a single message part
func (f *EmailFetcher) processPart(part *multipart.Part, emailMsg *EmailMessage) error {
	// Check if this is a PDF attachment (legacy support)
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
	} else {
		// Look for PDF links in text content
		return f.extractPDFLinksFromPart(part, emailMsg)
	}

	return nil
}

// extractPDFLinks extracts PDF download links from email content
func (f *EmailFetcher) extractPDFLinks(entity *mail.Message, emailMsg *EmailMessage) error {
	// Read the body content
	body, err := io.ReadAll(entity.Body)
	if err != nil {
		return fmt.Errorf("failed to read email body: %w", err)
	}

	bodyStr := string(body)

	// Look for Statstidende PDF links
	pdfLinks := f.findStatstidendePDFLinks(bodyStr)

	for _, link := range pdfLinks {
		// Check if this link has already been processed
		if emailMsg.processedLinks[link] {
			log.Printf("Skipping duplicate PDF link: %s", link)
			continue
		}
		emailMsg.processedLinks[link] = true

		log.Printf("Found PDF link: %s", link)

		// Add URL to the PDFURLs slice instead of downloading
		emailMsg.PDFURLs = append(emailMsg.PDFURLs, link)
	}

	return nil
}

// extractPDFLinksFromPart extracts PDF download links from a message part
func (f *EmailFetcher) extractPDFLinksFromPart(part *multipart.Part, emailMsg *EmailMessage) error {
	// Read the part content
	body, err := io.ReadAll(part)
	if err != nil {
		return fmt.Errorf("failed to read part body: %w", err)
	}

	bodyStr := string(body)

	// Look for Statstidende PDF links
	pdfLinks := f.findStatstidendePDFLinks(bodyStr)

	for _, link := range pdfLinks {
		// Check if this link has already been processed
		if emailMsg.processedLinks[link] {
			log.Printf("Skipping duplicate PDF link: %s", link)
			continue
		}
		emailMsg.processedLinks[link] = true

		log.Printf("Found PDF link: %s", link)

		// Add URL to the PDFURLs slice instead of downloading
		emailMsg.PDFURLs = append(emailMsg.PDFURLs, link)
	}

	return nil
}

// findStatstidendePDFLinks finds PDF download links in email content
func (f *EmailFetcher) findStatstidendePDFLinks(content string) []string {
	var links []string

	// Pattern for Statstidende PDF links
	// Looking for links like: https://statstidende.dk/api/publication/3093/pdf
	statstidendePattern := regexp.MustCompile(`https://statstidende\.dk/api/publication/\d+/pdf`)

	matches := statstidendePattern.FindAllString(content, -1)
	for _, match := range matches {
		links = append(links, match)
	}

	return links
}

// downloadPDF downloads a PDF from a URL
func (f *EmailFetcher) downloadPDF(url string) ([]byte, error) {
	log.Printf("Downloading PDF from: %s", url)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Make the request
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to download PDF: %w", err)
	}
	defer resp.Body.Close()

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download PDF: HTTP %d", resp.StatusCode)
	}

	// Check if the response is actually a PDF
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/pdf") {
		log.Printf("Warning: Response is not a PDF (Content-Type: %s)", contentType)
	}

	// Read the PDF data
	pdfData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read PDF data: %w", err)
	}

	log.Printf("Successfully downloaded PDF (%d bytes)", len(pdfData))
	return pdfData, nil
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
