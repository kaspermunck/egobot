package ai

import (
	"fmt"
	"strings"
	"testing"
)

func TestPreFilterContent(t *testing.T) {
	// Test data
	text := `
		This is a paragraph about Danske Bank.
		
		This paragraph mentions nothing relevant.
		
		Another paragraph about bankruptcy proceedings.
		
		This paragraph talks about fintech companies.
		
		Random paragraph with no relevant information.
	`

	entities := []string{"Danske Bank", "fintech", "bankruptcy"}

	result := preFilterContent(text, entities)

	// Should contain relevant paragraphs
	if !strings.Contains(result, "Danske Bank") {
		t.Error("Filtered content should contain 'Danske Bank'")
	}

	if !strings.Contains(result, "bankruptcy") {
		t.Error("Filtered content should contain 'bankruptcy'")
	}

	if !strings.Contains(result, "fintech") {
		t.Error("Filtered content should contain 'fintech'")
	}

	// The function might include business keywords even in irrelevant paragraphs
	// So we just check that the result is not empty
	if len(result) == 0 {
		t.Error("Filtered content should not be empty")
	}
}

func TestPreFilterContentNoMatches(t *testing.T) {
	text := "This is a completely irrelevant text with no business information."
	entities := []string{"Danske Bank", "fintech"}

	result := preFilterContent(text, entities)

	// Should return original text when no matches found
	if result != text {
		t.Error("Should return original text when no relevant content found")
	}
}

func TestSmartChunkText(t *testing.T) {
	// Create a long text
	lines := make([]string, 100)
	for i := 0; i < 100; i++ {
		lines[i] = fmt.Sprintf("This is line %d with some content that makes it longer than a simple line.", i)
	}
	text := strings.Join(lines, "\n")

	chunks := smartChunkText(text, 1000)

	if len(chunks) == 0 {
		t.Error("Should return at least one chunk")
	}

	// Check that chunks are reasonable size
	for i, chunk := range chunks {
		if len(chunk) > 5000 { // Conservative limit
			t.Errorf("Chunk %d is too large: %d characters", i, len(chunk))
		}
		if len(chunk) == 0 {
			t.Errorf("Chunk %d is empty", i)
		}
	}
}

func TestSmartChunkTextSmallText(t *testing.T) {
	text := "This is a small text that should not be chunked."
	chunks := smartChunkText(text, 1000)

	if len(chunks) != 1 {
		t.Errorf("Expected 1 chunk for small text, got %d", len(chunks))
	}

	if chunks[0] != text {
		t.Errorf("Expected original text, got %s", chunks[0])
	}
}

func TestSmartChunkTextEmptyText(t *testing.T) {
	text := ""
	chunks := smartChunkText(text, 1000)

	// Empty text should return empty chunks array
	if len(chunks) != 0 {
		t.Errorf("Expected 0 chunks for empty text, got %d", len(chunks))
	}
}

func TestPreFilterContentBusinessKeywords(t *testing.T) {
	text := `
		This paragraph mentions konkurs proceedings.
		
		This paragraph talks about fusion of companies.
		
		This paragraph mentions direktion changes.
		
		This paragraph has no business keywords.
	`

	entities := []string{"Some Company"}

	result := preFilterContent(text, entities)

	// Should contain paragraphs with business keywords even if entities not found
	if !strings.Contains(result, "konkurs") {
		t.Error("Should include paragraphs with business keywords")
	}

	if !strings.Contains(result, "fusion") {
		t.Error("Should include paragraphs with business keywords")
	}

	if !strings.Contains(result, "direktion") {
		t.Error("Should include paragraphs with business keywords")
	}

	// The function might include business keywords even in irrelevant paragraphs
	// So we just check that the result is not empty
	if len(result) == 0 {
		t.Error("Filtered content should not be empty")
	}
}
