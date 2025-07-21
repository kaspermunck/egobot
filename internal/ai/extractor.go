package ai

import (
	"context"
)

type ExtractionResult struct {
	Companies  []string `json:"companies"`
	VATNumbers []string `json:"vat_numbers"`
	Persons    []string `json:"persons"`
}

// ExtractEntitiesFromText is a placeholder for AI extraction logic.
// In production, this would call an external AI API (e.g., OpenAI, Gemini).
func ExtractEntitiesFromText(ctx context.Context, text string) (*ExtractionResult, error) {
	// TODO: Integrate with real AI API
	// For now, return dummy data
	return &ExtractionResult{
		Companies:  []string{"Example Company"},
		VATNumbers: []string{"12345678"},
		Persons:    []string{"John Doe"},
	}, nil
}
