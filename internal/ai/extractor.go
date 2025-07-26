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
			"konkurs", "bankruptcy", "liquidation", "insolvency",
			"fusion", "merger", "acquisition", "overtagelse",
			"stiftelse", "foundation", "oprettelse", "establishment",
			"ophør", "termination", "lukning", "closure",
			"ændring", "change", "modification",
			"direktion", "management", "bestyrelse", "board",
			"kapital", "capital", "aktier", "shares",
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

// ExtractEntitiesFromPDFFile extracts text from PDF and uses Chat Completions API to analyze it.
func ExtractEntitiesFromPDFFile(ctx context.Context, file io.Reader, filename string, entities []string) (ExtractionResult, error) {
	log.Printf("Starting PDF extraction for file: %s, entities: %v", filename, entities)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}
	client := openai.NewClient(apiKey)

	// 1. Extract text from PDF
	log.Printf("Extracting text from PDF")
	pdfText, err := pdf.ExtractText(file)
	if err != nil {
		log.Printf("PDF text extraction error: %v", err)
		return nil, fmt.Errorf("failed to extract text from PDF: %w", err)
	}
	log.Printf("Extracted %d characters from PDF", len(pdfText))

	// 2. Pre-filter content to only include relevant sections
	log.Printf("Pre-filtering content for relevant information")
	filteredText := preFilterContent(pdfText, entities)
	log.Printf("Filtered text length: %d characters", len(filteredText))

	// 3. Split text into smaller chunks to avoid rate limits
	// Use 1000 tokens per chunk (more conservative)
	chunks := smartChunkText(filteredText, 1000)
	log.Printf("Split filtered text into %d chunks", len(chunks))

	// 4. Process each chunk with rate limiting
	entityList := strings.Join(entities, ", ")
	allResults := make(ExtractionResult)

	for i, chunk := range chunks {
		log.Printf("Processing chunk %d/%d (%d characters)", i+1, len(chunks), len(chunk))

		// Add rate limiting delay between requests
		if i > 0 {
			time.Sleep(2 * time.Second) // 2 second delay between requests
		}

		// Use a more focused prompt to reduce token usage
		userPrompt := fmt.Sprintf(`Analyze this Danish business document chunk for mentions of: %s.

For each entity, provide a brief summary of any relevant changes or events. If not mentioned, say "No information found."

Document:
%s`, entityList, chunk)

		resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
			MaxTokens: 500, // Limit response size
		})
		if err != nil {
			log.Printf("OpenAI chat completion error for chunk %d: %v", i+1, err)

			// If we hit rate limits, wait and retry
			if strings.Contains(err.Error(), "429") || strings.Contains(err.Error(), "rate limit") {
				log.Printf("Rate limit hit, waiting 60 seconds before retry...")
				time.Sleep(60 * time.Second)

				// Retry once
				resp, err = client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
					Model: openai.GPT4o,
					Messages: []openai.ChatCompletionMessage{
						{
							Role:    openai.ChatMessageRoleUser,
							Content: userPrompt,
						},
					},
					MaxTokens: 500,
				})
				if err != nil {
					log.Printf("Retry failed for chunk %d: %v", i+1, err)
					return nil, fmt.Errorf("failed to get chat completion for chunk %d after retry: %w", i+1, err)
				}
			} else {
				return nil, fmt.Errorf("failed to get chat completion for chunk %d: %w", i+1, err)
			}
		}

		if len(resp.Choices) == 0 {
			log.Printf("No response for chunk %d", i+1)
			continue
		}

		answer := resp.Choices[0].Message.Content
		log.Printf("Received answer for chunk %d, length: %d", i+1, len(answer))

		// Parse results from this chunk
		for _, entity := range entities {
			if idx := strings.Index(strings.ToLower(answer), strings.ToLower(entity)); idx != -1 {
				rest := answer[idx:]
				end := strings.Index(rest, "\n\n")
				if end == -1 {
					end = len(rest)
				}
				entityInfo := strings.TrimSpace(rest[:end])

				// Append to existing result or create new one
				if existing, exists := allResults[entity]; exists && existing != "No information found." {
					allResults[entity] = existing + "\n\n" + entityInfo
				} else {
					allResults[entity] = entityInfo
				}
			}
		}
	}

	// 5. Fill in missing entities
	for _, entity := range entities {
		if _, exists := allResults[entity]; !exists {
			allResults[entity] = "No information found."
		}
	}

	log.Printf("Extraction completed, found info for %d entities", len(allResults))
	return allResults, nil
}
