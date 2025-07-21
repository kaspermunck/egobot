package ai

import (
	"context"
)

// ExtractionResult maps each entity to its extracted information.
type ExtractionResult map[string]string

// ExtractEntities takes a context, the PDF text, and a list of entities (person, VAT, company, industry, etc.) to extract info about.
// In production, this would call an external AI API (e.g., OpenAI, Gemini).
func ExtractEntities(ctx context.Context, text string, entities []string) (ExtractionResult, error) {
	// TODO: Integrate with real AI API
	// For now, return dummy data for each entity
	result := make(ExtractionResult)
	for _, entity := range entities {
		result[entity] = "Sample extracted info for: " + entity
	}
	return result, nil
}
