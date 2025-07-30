package processor

import (
	"context"
	"fmt"
	"testing"
	"time"

	"egobot/internal/ai"
	"egobot/internal/config"
	"egobot/internal/email"
)

// MockEmailFetcher for testing
type MockEmailFetcher struct {
	emails []email.EmailMessage
	err    error
}

func (m *MockEmailFetcher) FetchPDFEmails() ([]email.EmailMessage, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.emails, nil
}

// MockEmailSender for testing
type MockEmailSender struct {
	sentResults []email.AnalysisResult
	err         error
}

func (m *MockEmailSender) SendAnalysisResults(results []email.AnalysisResult) error {
	if m.err != nil {
		return m.err
	}
	m.sentResults = results
	return nil
}

func (m *MockEmailSender) SendErrorNotification(errorMsg string) error {
	return m.err
}

// MockExtractor for testing
type MockExtractor struct {
	results ai.ExtractionResult
	err     error
}

func (m *MockExtractor) ExtractEntitiesFromPDFFile(ctx context.Context, file interface{}, filename string, entities []string) (ai.ExtractionResult, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.results, nil
}

func (m *MockExtractor) ExtractEntitiesFromPDFURL(ctx context.Context, pdfURL string, entities []string) (ai.ExtractionResponse, error) {
	if m.err != nil {
		return ai.ExtractionResponse{}, m.err
	}
	return ai.ExtractionResponse{
		Results:     m.results,
		RawResponse: "Mock raw response for testing",
	}, nil
}

func TestNewProcessor(t *testing.T) {
	cfg := &config.Config{
		IMAPServer:      "imap.test.com",
		IMAPPort:        993,
		IMAPUsername:    "test@example.com",
		IMAPPassword:    "password",
		IMAPFolder:      "INBOX",
		SMTPHost:        "smtp.test.com",
		SMTPPort:        587,
		SMTPUsername:    "test@example.com",
		SMTPPassword:    "password",
		SMTPFrom:        "from@example.com",
		SMTPTo:          "to@example.com",
		OpenAIStub:      true,
		EntitiesToTrack: []string{"test"},
	}

	proc := NewProcessor(cfg)
	if proc == nil {
		t.Error("Expected processor to be created")
	}

	if proc.config != cfg {
		t.Error("Expected config to be set correctly")
	}
}

func TestProcessor_ProcessEmails_NoEmails(t *testing.T) {
	cfg := &config.Config{
		EntitiesToTrack: []string{"test"},
	}

	proc := &Processor{
		config: cfg,
		fetcher: &MockEmailFetcher{
			emails: []email.EmailMessage{},
		},
		sender:    &MockEmailSender{},
		extractor: &MockExtractor{},
	}

	err := proc.ProcessEmails()
	if err != nil {
		t.Errorf("Expected no error when no emails found, got %v", err)
	}
}

func TestProcessor_ProcessEmails_WithEmails(t *testing.T) {
	cfg := &config.Config{
		EntitiesToTrack: []string{"test", "example"},
	}

	mockFetcher := &MockEmailFetcher{
		emails: []email.EmailMessage{
			{
				ID:      "1",
				Subject: "Test Email",
				From:    "sender@example.com",
				Date:    time.Now(),
				PDFURLs: []string{"https://example.com/test.pdf"},
			},
		},
	}

	mockSender := &MockEmailSender{}

	mockExtractor := &MockExtractor{
		results: ai.ExtractionResult{
			"test":    "Test entity found",
			"example": "Example entity found",
		},
	}

	proc := &Processor{
		config:    cfg,
		fetcher:   mockFetcher,
		sender:    mockSender,
		extractor: mockExtractor,
	}

	err := proc.ProcessEmails()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if len(mockSender.sentResults) != 1 {
		t.Errorf("Expected 1 result to be sent, got %d", len(mockSender.sentResults))
	}

	result := mockSender.sentResults[0]
	if result.Filename != "statstidende.pdf" {
		t.Errorf("Expected filename 'statstidende.pdf', got %s", result.Filename)
	}

	if len(result.Entities) != 2 {
		t.Errorf("Expected 2 entities, got %d", len(result.Entities))
	}
}

func TestProcessor_ProcessEmails_ExtractionError(t *testing.T) {
	cfg := &config.Config{
		EntitiesToTrack: []string{"test"},
	}

	mockFetcher := &MockEmailFetcher{
		emails: []email.EmailMessage{
			{
				ID:      "1",
				Subject: "Test Email",
				From:    "sender@example.com",
				Date:    time.Now(),
				PDFURLs: []string{"https://example.com/test.pdf"},
			},
		},
	}

	mockSender := &MockEmailSender{}

	mockExtractor := &MockExtractor{
		err: fmt.Errorf("test error"),
	}

	proc := &Processor{
		config:    cfg,
		fetcher:   mockFetcher,
		sender:    mockSender,
		extractor: mockExtractor,
	}

	err := proc.ProcessEmails()
	if err != nil {
		t.Errorf("Expected no error (errors should be handled per attachment), got %v", err)
	}

	if len(mockSender.sentResults) != 1 {
		t.Errorf("Expected 1 result to be sent (with error), got %d", len(mockSender.sentResults))
	}

	result := mockSender.sentResults[0]
	if result.Error == "" {
		t.Error("Expected error to be set in result")
	}
}
