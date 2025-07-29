package email

import (
	"strings"
	"testing"
	"time"

	"egobot/internal/ai"
)

func TestNewEmailSender(t *testing.T) {
	config := &SenderConfig{
		Host:     "smtp.gmail.com",
		Port:     587,
		Username: "test@example.com",
		Password: "password123",
		From:     "from@example.com",
		To:       "to@example.com",
	}

	sender := NewEmailSender(config)
	if sender == nil {
		t.Error("Expected sender to be created")
	}

	if sender.config != config {
		t.Error("Expected config to be set correctly")
	}
}

func TestEmailSender_CleanEntityResult(t *testing.T) {
	sender := NewEmailSender(&SenderConfig{})

	tests := []struct {
		entityName string
		result     string
		expected   string
	}{
		{
			entityName: "4720 Præstø",
			result:     "4720 Præstø Dødsdato: N/A Skifteret: Retten i Sønderborg",
			expected:   "Dødsdato: N/A Skifteret: Retten i Sønderborg",
		},
		{
			entityName: "CPR-nr.: 2704430690",
			result:     "CPR-nr.: 2704430690 Adresse: 4720 Præstø Dødsdato: N/A",
			expected:   "Adresse: 4720 Præstø Dødsdato: N/A",
		},
		{
			entityName: "Stephen Richard Grieves",
			result:     "Stephen Richard Grieves CPR-nr.: 2704430690 Adresse: 4720 Præstø",
			expected:   "CPR-nr.: 2704430690 Adresse: 4720 Præstø",
		},
		{
			entityName: "Højbjerg",
			result:     "Højbjerg Dødsdato: 25.06.2025 Skifteret: Retten i Sønderborg",
			expected:   "Dødsdato: 25.06.2025 Skifteret: Retten i Sønderborg",
		},
		{
			entityName: "Test Entity",
			result:     "Some other text that doesn't start with the entity",
			expected:   "Some other text that doesn't start with the entity",
		},
		{
			entityName: "Test Entity",
			result:     "Test Entity: Some information",
			expected:   "Some information",
		},
		{
			entityName: "Test Entity",
			result:     "Test Entity - Some information",
			expected:   "Some information",
		},
	}

	for i, test := range tests {
		cleaned := sender.cleanEntityResult(test.entityName, test.result)
		if cleaned != test.expected {
			t.Errorf("Test %d: Expected '%s', got '%s'", i+1, test.expected, cleaned)
		}
	}
}

func TestEmailSender_GenerateHTMLContent(t *testing.T) {
	sender := NewEmailSender(&SenderConfig{})

	results := []AnalysisResult{
		{
			Filename:     "test1.pdf",
			EmailSubject: "Test Email 1",
			EmailFrom:    "sender1@example.com",
			EmailDate:    time.Now(),
			Entities: ai.ExtractionResult{
				"Danske Bank": "No significant changes reported.",
				"fintech":     "Several companies mentioned.",
			},
		},
		{
			Filename:     "test2.pdf",
			EmailSubject: "Test Email 2",
			EmailFrom:    "sender2@example.com",
			EmailDate:    time.Now(),
			Error:        "Failed to process PDF",
		},
	}

	htmlContent, err := sender.generateHTMLContent(results)
	if err != nil {
		t.Fatalf("Failed to generate HTML content: %v", err)
	}

	// Check that HTML contains expected content
	if !strings.Contains(htmlContent, "PDF Analysis Results") {
		t.Error("Expected HTML to contain 'PDF Analysis Results'")
	}

	if !strings.Contains(htmlContent, "test1.pdf") {
		t.Error("Expected HTML to contain first filename")
	}

	if !strings.Contains(htmlContent, "test2.pdf") {
		t.Error("Expected HTML to contain second filename")
	}

	if !strings.Contains(htmlContent, "Danske Bank") {
		t.Error("Expected HTML to contain entity name")
	}

	if !strings.Contains(htmlContent, "Failed to process PDF") {
		t.Error("Expected HTML to contain error message")
	}

	// Check for summary statistics
	if !strings.Contains(htmlContent, "Total PDFs processed: 2") {
		t.Error("Expected HTML to contain total count")
	}

	if !strings.Contains(htmlContent, "Successful analyses: 1") {
		t.Error("Expected HTML to contain success count")
	}

	if !strings.Contains(htmlContent, "Failed analyses: 1") {
		t.Error("Expected HTML to contain error count")
	}
}

func TestEmailSender_GenerateHTMLContent_EmptyResults(t *testing.T) {
	sender := NewEmailSender(&SenderConfig{})

	htmlContent, err := sender.generateHTMLContent([]AnalysisResult{})
	if err != nil {
		t.Fatalf("Failed to generate HTML content: %v", err)
	}

	if !strings.Contains(htmlContent, "PDF Analysis Results") {
		t.Error("Expected HTML to contain header")
	}

	if !strings.Contains(htmlContent, "Total PDFs processed: 0") {
		t.Error("Expected HTML to contain zero count")
	}
}

func TestEmailSender_SendAnalysisResults_EmptyResults(t *testing.T) {
	sender := NewEmailSender(&SenderConfig{})

	// This should not error even with empty results
	err := sender.SendAnalysisResults([]AnalysisResult{})
	if err != nil {
		t.Errorf("Expected no error with empty results, got %v", err)
	}
}
