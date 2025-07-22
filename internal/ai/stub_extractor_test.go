package ai

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestStubExtractor_ExtractEntitiesFromPDFFile(t *testing.T) {
	extractor := NewStubExtractor()
	ctx := context.Background()

	entities := []string{"Danske Bank", "fintech", "12345678"}

	start := time.Now()
	result, err := extractor.ExtractEntitiesFromPDFFile(ctx, nil, "test.pdf", entities)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that we got results for all entities
	if len(result) != len(entities) {
		t.Errorf("Expected %d results, got %d", len(entities), len(result))
	}

	// Check specific responses
	if !strings.Contains(result["Danske Bank"], "Danske Bank") {
		t.Error("Expected Danske Bank response to contain entity name")
	}
	if !strings.Contains(result["fintech"], "fintech") {
		t.Error("Expected fintech response to contain entity name")
	}
	if !strings.Contains(result["12345678"], "12345678") {
		t.Error("Expected VAT response to contain entity name")
	}

	// Check that processing took some time (simulated)
	if duration < 50*time.Millisecond {
		t.Error("Expected processing to take at least 50ms")
	}
}

func TestStubExtractor_ExtractEntitiesFromText(t *testing.T) {
	extractor := NewStubExtractor()
	ctx := context.Background()

	text := "This document contains information about Danske Bank and fintech companies."
	entities := []string{"Danske Bank", "fintech", "nonexistent"}

	result, err := extractor.ExtractEntitiesFromText(ctx, text, entities)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Check that we got results for all entities
	if len(result) != len(entities) {
		t.Errorf("Expected %d results, got %d", len(entities), len(result))
	}

	// Check that entities found in text have appropriate responses
	if !strings.Contains(result["Danske Bank"], "Found mentions") {
		t.Error("Expected Danske Bank to be marked as found")
	}
	if !strings.Contains(result["fintech"], "Found mentions") {
		t.Error("Expected fintech to be marked as found")
	}
	if !strings.Contains(result["nonexistent"], "No specific mentions") {
		t.Error("Expected nonexistent entity to be marked as not found")
	}
}

func TestStubExtractor_RealisticResponses(t *testing.T) {
	extractor := NewStubExtractor()
	ctx := context.Background()

	testCases := []struct {
		entity     string
		shouldFind bool
	}{
		{"Danske Bank", true},
		{"fintech", true},
		{"bankruptcy", true},
		{"12345678", true},
		{"John Doe", true},
		{"Random Company", false},
	}

	for _, tc := range testCases {
		result, err := extractor.ExtractEntitiesFromPDFFile(ctx, nil, "test.pdf", []string{tc.entity})

		if err != nil {
			t.Errorf("Error processing entity %s: %v", tc.entity, err)
			continue
		}

		if len(result) != 1 {
			t.Errorf("Expected 1 result for %s, got %d", tc.entity, len(result))
			continue
		}

		response := result[tc.entity]
		if tc.shouldFind && !strings.Contains(response, tc.entity) {
			t.Errorf("Expected response for %s to contain entity name", tc.entity)
		}
	}
}
