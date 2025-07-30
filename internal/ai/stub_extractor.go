package ai

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

// StubExtractor provides fake but realistic responses for testing
type StubExtractor struct{}

// NewStubExtractor creates a new stub extractor
func NewStubExtractor() *StubExtractor {
	return &StubExtractor{}
}

// ExtractEntitiesFromPDFFile provides stubbed responses for testing
func (s *StubExtractor) ExtractEntitiesFromPDFFile(ctx context.Context, file interface{}, filename string, entities []string) (ExtractionResult, error) {
	log.Printf("STUB: Processing PDF file: %s with entities: %v", filename, entities)

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Generate realistic fake responses based on entities
	result := make(ExtractionResult)

	for _, entity := range entities {
		entityLower := strings.ToLower(entity)

		switch {
		case strings.Contains(entityLower, "danske"):
			result[entity] = "Danske Bank: No significant changes reported in this document. The bank continues normal operations."
		case strings.Contains(entityLower, "fintech"):
			result[entity] = "Fintech: Several fintech companies mentioned in regulatory updates. New compliance requirements for digital payment services."
		case strings.Contains(entityLower, "bankruptcy"):
			result[entity] = "Bankruptcy: Three companies filed for bankruptcy protection this period. All cases are under court supervision."
		case strings.Contains(entityLower, "12345678"):
			result[entity] = "VAT 12345678: Company with this VAT number has updated their board composition. New CEO appointed effective next month."
		case strings.Contains(entityLower, "john doe"):
			result[entity] = "John Doe: No specific mentions found for this individual in the current document."
		default:
			result[entity] = fmt.Sprintf("%s: Limited information available. Recommend checking previous documents for historical data.", entity)
		}
	}

	// Add a summary if no specific entities found
	if len(result) == 0 {
		result["summary"] = "Document processed successfully. No specific entities matched the search criteria."
	}

	log.Printf("STUB: Generated results for %d entities", len(result))
	return result, nil
}

// ExtractEntitiesFromPDFURL provides stubbed responses for URL-based PDF analysis
func (s *StubExtractor) ExtractEntitiesFromPDFURL(ctx context.Context, pdfURL string, entities []string) (ExtractionResponse, error) {
	log.Printf("STUB: Processing PDF URL: %s with entities: %v", pdfURL, entities)

	// Simulate processing time
	time.Sleep(100 * time.Millisecond)

	// Generate realistic fake responses based on entities
	result := make(ExtractionResult)

	for _, entity := range entities {
		entityLower := strings.ToLower(entity)

		switch {
		case strings.Contains(entityLower, "danske"):
			result[entity] = "Danske Bank: No significant changes reported in this document. The bank continues normal operations."
		case strings.Contains(entityLower, "fintech"):
			result[entity] = "Fintech: Several fintech companies mentioned in regulatory updates. New compliance requirements for digital payment services."
		case strings.Contains(entityLower, "bankruptcy"):
			result[entity] = "Bankruptcy: Three companies filed for bankruptcy protection this period. All cases are under court supervision."
		case strings.Contains(entityLower, "12345678"):
			result[entity] = "VAT 12345678: Company with this VAT number has updated their board composition. New CEO appointed effective next month."
		case strings.Contains(entityLower, "john doe"):
			result[entity] = "John Doe: No specific mentions found for this individual in the current document."
		default:
			result[entity] = fmt.Sprintf("%s: Limited information available. Recommend checking previous documents for historical data.", entity)
		}
	}

	// Add a summary if no specific entities found
	if len(result) == 0 {
		result["summary"] = "Document processed successfully. No specific entities matched the search criteria."
	}

	// Create a realistic raw response
	rawResponse := "Her er den relevante information:\n\n"
	for entity, info := range result {
		rawResponse += fmt.Sprintf("### %s\n%s\n\n", entity, info)
	}

	log.Printf("STUB: Generated results for %d entities from URL", len(result))
	return ExtractionResponse{
		Results:     result,
		RawResponse: rawResponse,
	}, nil
}

// ExtractEntitiesFromText provides stubbed responses for text analysis
func (s *StubExtractor) ExtractEntitiesFromText(ctx context.Context, text string, entities []string) (ExtractionResult, error) {
	log.Printf("STUB: Processing text (%d chars) with entities: %v", len(text), entities)

	// Simulate processing time
	time.Sleep(50 * time.Millisecond)

	// Generate responses based on text content and entities
	result := make(ExtractionResult)

	for _, entity := range entities {
		entityLower := strings.ToLower(entity)
		textLower := strings.ToLower(text)

		if strings.Contains(textLower, entityLower) {
			result[entity] = fmt.Sprintf("%s: Found mentions in document. Analysis indicates normal business activities.", entity)
		} else {
			result[entity] = fmt.Sprintf("%s: No specific mentions found in this document section.", entity)
		}
	}

	if len(result) == 0 {
		result["summary"] = "Text analyzed successfully. No specific entities found in this section."
	}

	return result, nil
}
