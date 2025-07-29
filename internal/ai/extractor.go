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

// extractRelevantSections extracts only sections that contain the target entities
func extractRelevantSections(text string, entities []string) string {
	// Split text into sentences for more granular filtering
	sentences := strings.Split(text, ". ")
	var relevantSentences []string

	// Track which entities we've found
	foundEntities := make(map[string]bool)

	for _, sentence := range sentences {
		sentenceLower := strings.ToLower(sentence)

		// Check if sentence contains any of the target entities
		for _, entity := range entities {
			if findEntityInText(sentence, entity) {
				relevantSentences = append(relevantSentences, sentence)
				foundEntities[entity] = true
				break
			}
		}

		// Also include sentences with business keywords
		// We include sentences containing business keywords (like bankruptcy, death estate, etc.)
		// because they often provide important context about the entities we're tracking, even if
		// the entities aren't directly mentioned in those sentences. For example, a sentence with
		// "dødsbo" (death estate) might explain what happened to a person we're tracking, while
		// a sentence with "konkurs" (bankruptcy) might explain what happened to their business.
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

	// If we found relevant content, return it
	if len(relevantSentences) > 0 {
		filteredText := strings.Join(relevantSentences, ". ")
		log.Printf("Section extraction: found %d relevant sentences", len(relevantSentences))
		return filteredText
	}

	// If no relevant content found, return first 1000 characters
	log.Printf("No relevant sections found, using first 1000 characters")
	if len(text) > 1000 {
		return text[:1000]
	}
	return text
}

// truncateTextToTokenLimit truncates text to fit within token limits
func truncateTextToTokenLimit(text string, maxTokens int) string {
	// Rough estimate: 4 characters per token
	maxChars := maxTokens * 3 // Conservative estimate

	if len(text) <= maxChars {
		return text
	}

	// Try to truncate at sentence boundaries
	sentences := strings.Split(text, ". ")
	if len(sentences) == 1 {
		// No sentence boundaries, truncate directly
		return text[:maxChars] + "..."
	}

	var result strings.Builder
	charCount := 0

	for _, sentence := range sentences {
		sentenceWithPeriod := sentence + ". "
		if charCount+len(sentenceWithPeriod) > maxChars {
			break
		}
		result.WriteString(sentenceWithPeriod)
		charCount += len(sentenceWithPeriod)
	}

	if result.Len() == 0 {
		// If we couldn't fit even one sentence, truncate directly
		return text[:maxChars] + "..."
	}

	return strings.TrimSpace(result.String())
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

	// 4. Use aggressive sentence-level filtering to reduce token usage
	log.Printf("Entities found, extracting only relevant sentences to minimize token usage")
	textToProcess := extractRelevantSections(pdfText, entities)
	log.Printf("Sentence extraction result: %d characters", len(textToProcess))

	// 5. If still too long, use even more aggressive filtering
	if len(textToProcess) > 8000 { // Conservative limit for GPT-3.5-turbo
		log.Printf("Text still too long (%d chars), applying ultra-aggressive filtering", len(textToProcess))
		textToProcess = extractUltraRelevantContent(textToProcess, entities)
		log.Printf("Ultra-aggressive filtering result: %d characters", len(textToProcess))
	}

	// 6. Process with GPT-3.5-turbo
	log.Printf("Processing document with GPT-3.5-turbo (%d characters)", len(textToProcess))

	entityList := strings.Join(entities, "\n- ")
	if len(entityList) > 0 {
		entityList = "- " + entityList
	}
	allResults := make(ExtractionResult)

	// Use the Danish prompt for Statstidende analysis
	userPrompt := fmt.Sprintf(`Du er advokat med speciale i konkursboer, dødsboer og tvangsauktioner. Du forstår hvilken information der er relevant for hver type af sag. Analyser denne udgave af statstidende og find relevant info for de adresser (herunder postnumre, bynavne), personnavne, cpr-numre, virkosmhedsnavne, og cvr-numre, som jeg giver dig. Medtag udelukkende følgende information for hver sagstype:
- Dødsboer: navn, cpr, adresse, dødsdato
- Konkursboer: virksomhedsnavn, cvr, hvornår konkursbegæring er modtaget
- Tvangsauktioner: matrikel og/eller adresse på ejendom

Find relevant information for følgende:
%s

Betragt hvert af punkterne isoleret, de har ikke noget med hinanden at gøre og skal analyseres separat. Hvert punkt kan optræde flere gange (fx adresse der deles af virksomhed og person), medtag i de tilfælde alle matches.
	
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

// extractUltraRelevantContent extracts only the most relevant content containing the target entities
func extractUltraRelevantContent(text string, entities []string) string {
	// Split into sentences
	sentences := strings.Split(text, ". ")
	var ultraRelevantSentences []string

	// Only include sentences that directly contain the target entities
	for _, sentence := range sentences {
		for _, entity := range entities {
			if findEntityInText(sentence, entity) {
				ultraRelevantSentences = append(ultraRelevantSentences, sentence)
				break
			}
		}
	}

	// If we found sentences with entities, return them
	if len(ultraRelevantSentences) > 0 {
		result := strings.Join(ultraRelevantSentences, ". ")
		log.Printf("Ultra-aggressive filtering: found %d sentences with target entities", len(ultraRelevantSentences))
		return result
	}

	// If no sentences with entities found, return first 500 characters
	log.Printf("No sentences with target entities found, using first 500 characters")
	if len(text) > 500 {
		return text[:500]
	}
	return text
}
