package processor

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"egobot/internal/ai"
	"egobot/internal/config"
	"egobot/internal/email"
)

// Processor orchestrates the email fetching, PDF analysis, and result sending
type Processor struct {
	config    *config.Config
	fetcher   EmailFetcher
	sender    EmailSender
	extractor Extractor
}

// EmailFetcher interface for email fetching
type EmailFetcher interface {
	FetchPDFEmails() ([]email.EmailMessage, error)
}

// EmailSender interface for email sending
type EmailSender interface {
	SendAnalysisResults(results []email.AnalysisResult) error
	SendErrorNotification(errorMsg string) error
}

// Extractor interface for AI extraction (allows both real and stubbed implementations)
type Extractor interface {
	ExtractEntitiesFromPDFFile(ctx context.Context, file interface{}, filename string, entities []string) (ai.ExtractionResult, error)
	ExtractEntitiesFromPDFURL(ctx context.Context, pdfURL string, entities []string) (ai.ExtractionResult, error)
}

// NewProcessor creates a new email processor
func NewProcessor(config *config.Config) *Processor {
	// Create email fetcher
	fetcherConfig := &email.Config{
		Server:   config.IMAPServer,
		Port:     config.IMAPPort,
		Username: config.IMAPUsername,
		Password: config.IMAPPassword,
		Folder:   config.IMAPFolder,
	}
	fetcher := email.NewEmailFetcher(fetcherConfig)

	// Create email sender
	senderConfig := &email.SenderConfig{
		Host:     config.SMTPHost,
		Port:     config.SMTPPort,
		Username: config.SMTPUsername,
		Password: config.SMTPPassword,
		From:     config.SMTPFrom,
		To:       config.SMTPTo,
	}
	sender := email.NewEmailSender(senderConfig)

	// Create extractor (real or stubbed)
	var extractor Extractor
	if config.OpenAIStub {
		extractor = ai.NewStubExtractor()
		log.Printf("Using stubbed AI extractor for testing")
	} else {
		extractor = &RealExtractor{}
		log.Printf("Using real OpenAI extractor")
	}

	return &Processor{
		config:    config,
		fetcher:   fetcher,
		sender:    sender,
		extractor: extractor,
	}
}

// RealExtractor wraps the real AI extractor
type RealExtractor struct{}

func (r *RealExtractor) ExtractEntitiesFromPDFFile(ctx context.Context, file interface{}, filename string, entities []string) (ai.ExtractionResult, error) {
	// Convert interface{} to io.Reader for the real extractor
	if reader, ok := file.(io.Reader); ok {
		return ai.ExtractEntitiesFromPDFFile(ctx, reader, filename, entities)
	}
	return nil, fmt.Errorf("file is not an io.Reader")
}

func (r *RealExtractor) ExtractEntitiesFromPDFURL(ctx context.Context, pdfURL string, entities []string) (ai.ExtractionResult, error) {
	return ai.ExtractEntitiesFromPDFURL(ctx, pdfURL, entities)
}

// ProcessEmails fetches emails, analyzes PDFs, and sends results
func (p *Processor) ProcessEmails() error {
	log.Printf("Starting email processing at %s", time.Now().Format("2006-01-02 15:04:05"))

	// 1. Fetch emails with PDF URLs
	emailMessages, err := p.fetcher.FetchPDFEmails()
	if err != nil {
		log.Printf("Failed to fetch emails: %v", err)
		return fmt.Errorf("failed to fetch emails: %w", err)
	}

	if len(emailMessages) == 0 {
		log.Printf("No emails with PDF URLs found")
		return nil
	}

	log.Printf("Found %d emails with PDF URLs", len(emailMessages))

	// 2. Process each email and its PDF URLs
	var analysisResults []email.AnalysisResult
	for _, emailMsg := range emailMessages {
		log.Printf("Processing email: %s (from %s)", emailMsg.Subject, emailMsg.From)

		for _, pdfURL := range emailMsg.PDFURLs {
			result := p.processPDFURL(pdfURL, emailMsg)
			analysisResults = append(analysisResults, result)
		}
	}

	// 3. Send results email
	if len(analysisResults) > 0 {
		if err := p.sender.SendAnalysisResults(analysisResults); err != nil {
			log.Printf("Failed to send analysis results: %v", err)
			return fmt.Errorf("failed to send analysis results: %w", err)
		}
		log.Printf("Successfully sent analysis results for %d PDFs", len(analysisResults))
	}

	log.Printf("Email processing completed successfully")
	return nil
}

// processPDFURL processes a single PDF URL
func (p *Processor) processPDFURL(pdfURL string, emailMsg email.EmailMessage) email.AnalysisResult {
	result := email.AnalysisResult{
		Filename:     "statstidende.pdf", // Use a default filename since we're working with URLs
		EmailSubject: emailMsg.Subject,
		EmailFrom:    emailMsg.From,
		EmailDate:    emailMsg.Date,
	}

	log.Printf("Analyzing PDF from URL: %s", pdfURL)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Extract entities from PDF URL
	entities, err := p.extractor.ExtractEntitiesFromPDFURL(ctx, pdfURL, p.config.EntitiesToTrack)
	if err != nil {
		log.Printf("Failed to extract entities from %s: %v", pdfURL, err)
		result.Error = fmt.Sprintf("Failed to extract entities: %v", err)
		return result
	}

	result.Entities = entities
	log.Printf("Successfully extracted entities from %s", pdfURL)
	return result
}

// ProcessWithRetry processes emails with retry logic
func (p *Processor) ProcessWithRetry() error {
	var lastErr error

	for attempt := 1; attempt <= p.config.MaxRetries; attempt++ {
		log.Printf("Processing attempt %d/%d", attempt, p.config.MaxRetries)

		if err := p.ProcessEmails(); err != nil {
			lastErr = err
			log.Printf("Attempt %d failed: %v", attempt, err)

			if attempt < p.config.MaxRetries {
				log.Printf("Waiting %v before retry", p.config.RetryDelay)
				time.Sleep(p.config.RetryDelay)
			}
		} else {
			log.Printf("Processing succeeded on attempt %d", attempt)
			return nil
		}
	}

	// All attempts failed, send error notification
	if lastErr != nil {
		log.Printf("All processing attempts failed, sending error notification")
		if err := p.sender.SendErrorNotification(lastErr.Error()); err != nil {
			log.Printf("Failed to send error notification: %v", err)
		}
	}

	return fmt.Errorf("all %d processing attempts failed: %w", p.config.MaxRetries, lastErr)
}
