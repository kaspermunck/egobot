package ai

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

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

	// 2. Split text into chunks to avoid token limits
	// Use 2000 tokens per chunk (leaving room for prompt and response)
	chunks := chunkText(pdfText, 2000)
	log.Printf("Split PDF into %d chunks", len(chunks))

	// 3. Process each chunk and collect results
	entityList := strings.Join(entities, ", ")
	allResults := make(ExtractionResult)

	for i, chunk := range chunks {
		log.Printf("Processing chunk %d/%d (%d characters)", i+1, len(chunks), len(chunk))

		userPrompt := fmt.Sprintf(`Analyze the following Danish company registry document chunk and extract all relevant information for each of the following entities: %s. For each entity, return a short summary of any changes, events, or mentions (such as bankruptcy, acquisition, management change, etc). If an entity is not mentioned, say so.

Document chunk:
%s`, entityList, chunk)

		resp, err := client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: userPrompt,
				},
			},
		})
		if err != nil {
			log.Printf("OpenAI chat completion error for chunk %d: %v", i+1, err)
			return nil, fmt.Errorf("failed to get chat completion for chunk %d: %w", i+1, err)
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

	// 4. Fill in missing entities
	for _, entity := range entities {
		if _, exists := allResults[entity]; !exists {
			allResults[entity] = "No information found."
		}
	}

	log.Printf("Extraction completed, found info for %d entities", len(allResults))
	return allResults, nil
}
