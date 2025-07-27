package ai

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"egobot/internal/pdf"

	openai "github.com/sashabaranov/go-openai"
)

// ExtractionResult maps each entity to its extracted information.
type ExtractionResult map[string]string

// chunkText splits text into chunks of approximately maxTokens characters
func chunkText(text string, maxTokens int) []string {
	// Roughly 4 characters per token
	maxChars := maxTokens * 4
	if len(text) <= maxChars {
		return []string{text}
	}

	var chunks []string
	for len(text) > 0 {
		chunkSize := maxChars
		if chunkSize > len(text) {
			chunkSize = len(text)
		}

		chunk := text[:chunkSize]

		// Try to break at a sentence boundary
		if lastPeriod := strings.LastIndex(chunk, "."); lastPeriod > chunkSize*3/4 {
			chunk = chunk[:lastPeriod+1]
		}

		chunks = append(chunks, chunk)
		text = text[len(chunk):]
	}

	return chunks
}

// preFilterContent filters PDF content to only include relevant sections
func preFilterContent(text string, entities []string) string {
	// Convert entities to lowercase for case-insensitive matching
	entityPatterns := make([]string, len(entities))
	for i, entity := range entities {
		entityPatterns[i] = strings.ToLower(entity)
	}

	// Split text into paragraphs
	paragraphs := strings.Split(text, "\n\n")
	var relevantParagraphs []string

	for _, paragraph := range paragraphs {
		paragraphLower := strings.ToLower(paragraph)

		// Check if paragraph contains any of the entities
		for _, entity := range entityPatterns {
			if strings.Contains(paragraphLower, entity) {
				relevantParagraphs = append(relevantParagraphs, paragraph)
				break
			}
		}

		// Also include paragraphs that might contain relevant business information
		// Look for keywords that indicate business events
		businessKeywords := []string{
			"frivillig likvidation", "dødsbo", "konkurs", "tvangsauktion", "fusion",
		}

		for _, keyword := range businessKeywords {
			if strings.Contains(paragraphLower, keyword) {
				relevantParagraphs = append(relevantParagraphs, paragraph)
				break
			}
		}
	}

	// If we found relevant content, return it; otherwise return original text
	if len(relevantParagraphs) > 0 {
		filteredText := strings.Join(relevantParagraphs, "\n\n")
		log.Printf("Pre-filtered content: %d paragraphs -> %d relevant paragraphs", len(paragraphs), len(relevantParagraphs))
		return filteredText
	}

	log.Printf("No relevant content found, using original text")
	return text
}

// aggressivePreFilterContent filters PDF content more aggressively to reduce token usage
func aggressivePreFilterContent(text string, entities []string) string {
	// Convert entities to lowercase for case-insensitive matching
	entityPatterns := make([]string, len(entities))
	for i, entity := range entities {
		entityPatterns[i] = strings.ToLower(entity)
	}

	// Split text into sentences for more granular filtering
	sentences := strings.Split(text, ". ")
	var relevantSentences []string

	for _, sentence := range sentences {
		sentenceLower := strings.ToLower(sentence)

		// Check if sentence contains any of the entities
		for _, entity := range entityPatterns {
			if strings.Contains(sentenceLower, entity) {
				relevantSentences = append(relevantSentences, sentence)
				break
			}
		}

		// Also include sentences that might contain relevant business information
		businessKeywords := []string{
			"frivillig likvidation", "dødsbo", "konkurs", "tvangsauktion", "fusion",
			"skifteret", "sagsnummer", "cpr", "cvr", "adresse", "dødsdato",
		}

		for _, keyword := range businessKeywords {
			if strings.Contains(sentenceLower, keyword) {
				relevantSentences = append(relevantSentences, sentence)
				break
			}
		}
	}

	// If we found relevant content, return it; otherwise return a minimal version
	if len(relevantSentences) > 0 {
		filteredText := strings.Join(relevantSentences, ". ")
		log.Printf("Aggressive pre-filtered content: %d sentences -> %d relevant sentences", len(strings.Split(text, ". ")), len(relevantSentences))
		return filteredText
	}

	// If no relevant content found, return a minimal version with just the first 1000 characters
	log.Printf("No relevant content found, using minimal text")
	if len(text) > 1000 {
		return text[:1000] + "..."
	}
	return text
}

// smartChunkText splits text into smaller, more focused chunks
func smartChunkText(text string, maxTokens int) []string {
	// Handle empty text
	if len(strings.TrimSpace(text)) == 0 {
		return []string{}
	}

	// Use smaller chunks to avoid rate limits
	maxChars := maxTokens * 3 // More conservative estimate
	if len(text) <= maxChars {
		return []string{text}
	}

	var chunks []string
	lines := strings.Split(text, "\n")
	var currentChunk strings.Builder
	currentLength := 0

	for _, line := range lines {
		lineLength := len(line)

		// If adding this line would exceed the limit, start a new chunk
		if currentLength+lineLength > maxChars && currentLength > 0 {
			chunk := strings.TrimSpace(currentChunk.String())
			if chunk != "" {
				chunks = append(chunks, chunk)
			}
			currentChunk.Reset()
			currentLength = 0
		}

		currentChunk.WriteString(line)
		currentChunk.WriteString("\n")
		currentLength += lineLength + 1
	}

	// Add the last chunk
	if currentChunk.Len() > 0 {
		chunk := strings.TrimSpace(currentChunk.String())
		if chunk != "" {
			chunks = append(chunks, chunk)
		}
	}

	return chunks
}

// extractRelevantSections extracts only the relevant sections from the PDF based on actual structure
func extractRelevantSections(text string, entities []string) string {
	var relevantSections []string

	// Split by major sections (these are the actual section headers in Statstidende)
	sections := []string{
		"Dødsboer",
		"Gældssanering",
		"Konkursboer",
		"Tvangsauktioner",
		"Øvrige retslige kundgørelser",
	}

	// Extract each relevant section
	for _, section := range sections {
		if idx := strings.Index(text, section); idx != -1 {
			// Find the end of this section (next section or end of text)
			end := len(text)
			for _, nextSection := range sections {
				if nextIdx := strings.Index(text[idx+len(section):], nextSection); nextIdx != -1 {
					if idx+len(section)+nextIdx < end {
						end = idx + len(section) + nextIdx
					}
				}
			}

			sectionContent := text[idx:end]
			relevantSections = append(relevantSections, sectionContent)
		}
	}

	// If we found sections, return them
	if len(relevantSections) > 0 {
		result := strings.Join(relevantSections, "\n\n")
		log.Printf("Extracted %d relevant sections", len(relevantSections))
		return result
	}

	// If no sections found, check if any entities are in the text
	// If entities are found, return the full text to ensure we don't miss anything
	for _, entity := range entities {
		if strings.Contains(strings.ToLower(text), strings.ToLower(entity)) {
			log.Printf("Entity '%s' found in text, using full document", entity)
			return text
		}
	}

	// If no sections and no entities found, return a larger portion of the text
	log.Printf("No relevant sections or entities found, using first 10000 characters")
	if len(text) > 10000 {
		return text[:10000]
	}
	return text
}

// findEntityInText performs robust entity matching with various strategies
func findEntityInText(text, entity string) bool {
	// Normalize both text and entity for comparison
	normalizedText := strings.ToLower(strings.ReplaceAll(text, " ", ""))
	normalizedEntity := strings.ToLower(strings.ReplaceAll(entity, " ", ""))

	// Strategy 1: Direct substring match
	if strings.Contains(normalizedText, normalizedEntity) {
		return true
	}

	// Strategy 2: Split entity into parts and check each part
	entityParts := strings.Fields(entity)
	if len(entityParts) > 1 {
		allPartsFound := true
		for _, part := range entityParts {
			if !strings.Contains(strings.ToLower(text), strings.ToLower(part)) {
				allPartsFound = false
				break
			}
		}
		if allPartsFound {
			return true
		}
	}

	// Strategy 3: Check for common variations (e.g., "0605410146" vs "06 05 41 01 46")
	// Remove spaces and special characters for comparison
	cleanText := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(text, " ", ""), "-", ""), ".", "")
	cleanEntity := strings.ReplaceAll(strings.ReplaceAll(strings.ReplaceAll(entity, " ", ""), "-", ""), ".", "")

	if strings.Contains(strings.ToLower(cleanText), strings.ToLower(cleanEntity)) {
		return true
	}

	// Strategy 4: Check for partial matches (for addresses)
	if len(entity) > 5 {
		// For longer entities like addresses, check if major parts are present
		words := strings.Fields(entity)
		if len(words) >= 2 {
			// Check if at least 2 words from the entity are found
			foundWords := 0
			for _, word := range words {
				if len(word) > 2 && strings.Contains(strings.ToLower(text), strings.ToLower(word)) {
					foundWords++
				}
			}
			if foundWords >= 2 {
				return true
			}
		}
	}

	return false
}

// ExtractEntitiesFromPDFFile uses comprehensive document processing with early termination
func ExtractEntitiesFromPDFFile(ctx context.Context, file io.Reader, filename string, entities []string) (ExtractionResult, error) {
	log.Printf("Starting PDF analysis for file: %s, entities: %v", filename, entities)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// 1. Extract text from PDF
	log.Printf("Extracting text from PDF")
	pdfText, err := pdf.ExtractText(file)
	if err != nil {
		log.Printf("PDF text extraction error: %v", err)
		return nil, fmt.Errorf("failed to extract text from PDF: %w", err)
	}
	log.Printf("Extracted %d characters from PDF", len(pdfText))

	// 2. Check if entities are in the text using robust matching
	log.Printf("Checking if entities are present in the document")
	entitiesFound := false
	for _, entity := range entities {
		if findEntityInText(pdfText, entity) {
			log.Printf("Entity '%s' found in document", entity)
			entitiesFound = true
			break
		} else {
			log.Printf("Entity '%s' NOT found in document", entity)
		}
	}

	// 3. Early termination if no entities found
	if !entitiesFound {
		log.Printf("No entities found in document, returning early with 'No information found' for all entities")
		allResults := make(ExtractionResult)
		for _, entity := range entities {
			allResults[entity] = "No information found."
		}
		return allResults, nil
	}

	// 4. Use different strategies based on document size and entity presence
	var textToProcess string
	if entitiesFound {
		// If entities are found, use the full document to ensure we capture everything
		log.Printf("Entities found, using full document (%d characters)", len(pdfText))
		textToProcess = pdfText
	} else {
		// If no entities found, try section extraction first
		log.Printf("No entities found, trying section extraction")
		textToProcess = extractRelevantSections(pdfText, entities)
		log.Printf("Section extraction result: %d characters", len(textToProcess))
	}

	// 5. Process with GPT-3.5-turbo (higher rate limits)
	log.Printf("Processing document with GPT-3.5-turbo (%d characters)", len(textToProcess))

	entityList := strings.Join(entities, "\n")
	allResults := make(ExtractionResult)

	// Use the Danish prompt for Statstidende analysis
	userPrompt := fmt.Sprintf(`I Statstidende optages alle meddelelser, som i henhold til lovgivningen skal kundgøres, herunder dødsboer, tvangsauktioner, gældssanering m.m. Du skal ekstrahere og opsummere alle relevante oplysninger ud fra følgende nøgleord:

%s

Vigtigt: Medtag også forekomster, hvor nøgleordene kun optræder som en del af en adresse (f.eks. et postnummer eller bynavn nævnt alene). Oplys kortfattet navn, CPR/CVR, adresse, dødsdato (hvis relevant), skifteret/sagsnummer og evt. behandlingstype (f.eks. insolvent/§ 69), samt hvordan og hvornår krav skal anmeldes.

Dokument:
%s`, entityList, textToProcess)

	// Use GPT-3.5-turbo (higher rate limits: 90k TPM vs 30k TPM)
	client := openai.NewClient(apiKey)

	// Initial delay
	delay := 1 * time.Second
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Attempt %d/%d, waiting %v before retry...", attempt+1, maxRetries, delay)
			time.Sleep(delay)
		}

		resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model: openai.GPT3Dot5Turbo,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
			MaxTokens: 4000,
		})

		if err != nil {
			log.Printf("OpenAI chat completion error (attempt %d): %v", attempt+1, err)

			// Check if it's a rate limit error
			if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit") || strings.Contains(err.Error(), "Too Many Requests") {
				if attempt < maxRetries-1 {
					// Exponential backoff: double the delay for next attempt
					delay = delay * 2
					if delay > 60*time.Second {
						delay = 60 * time.Second // Cap at 60 seconds
					}
					continue
				} else {
					return nil, fmt.Errorf("rate limit exceeded after %d retries: %w", maxRetries, err)
				}
			} else {
				return nil, fmt.Errorf("failed to get chat completion: %w", err)
			}
		}

		if len(resp.Choices) == 0 {
			log.Printf("No response from OpenAI")
			return nil, fmt.Errorf("no response from OpenAI")
		}

		answer := resp.Choices[0].Message.Content
		log.Printf("Received answer, length: %d", len(answer))

		// Parse results - look for each entity in the response
		for _, entity := range entities {
			if idx := strings.Index(strings.ToLower(answer), strings.ToLower(entity)); idx != -1 {
				rest := answer[idx:]
				end := strings.Index(rest, "\n\n")
				if end == -1 {
					end = len(rest)
				}
				entityInfo := strings.TrimSpace(rest[:end])

				// Set the result for this entity
				allResults[entity] = entityInfo
			} else {
				// Entity not found in response
				allResults[entity] = "No information found."
			}
		}

		log.Printf("Extraction completed, found info for %d entities", len(allResults))
		return allResults, nil
	}

	return nil, fmt.Errorf("failed to process document after %d attempts", maxRetries)
}
