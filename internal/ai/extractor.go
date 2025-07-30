package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// ExtractionResult maps each entity to its extracted information.
type ExtractionResult map[string]string

// ExtractionResponse contains both the parsed results and the raw OpenAI response
type ExtractionResponse struct {
	Results     ExtractionResult
	RawResponse string
}

// ExtractEntitiesFromPDFURL uses OpenAI's file_url parameter to analyze PDFs directly from URLs
func ExtractEntitiesFromPDFURL(ctx context.Context, pdfURL string, entities []string) (ExtractionResponse, error) {
	log.Printf("Starting PDF analysis for URL: %s", pdfURL)

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return ExtractionResponse{}, fmt.Errorf("OPENAI_API_KEY environment variable not set")
	}

	// Create the entity list for the prompt
	entityList := strings.Join(entities, "\n- ")
	if len(entityList) > 0 {
		entityList = "- " + entityList
	}

	log.Printf("Entities to look for: \n%s", entityList)

	// Use the Danish prompt for Statstidende analysis
	userPrompt := fmt.Sprintf(`Du er advokat med speciale i konkursboer, dødsboer og tvangsauktioner. Du forstår hvilken information der er relevant for hver type af sag. Analyser denne udgave af statstidende og find relevant info for de adresser (herunder postnumre, bynavne), personnavne, cpr-numre, virkosmhedsnavne, og cvr-numre, som jeg giver dig. Medtag udelukkende følgende information for hver sagstype:
	- Dødsboer: navn, cpr, adresse, dødsdato
	- Konkursboer: virksomhedsnavn, cvr, hvornår konkursbegæring er modtaget
	- Tvangsauktioner: matrikel og/eller adresse på ejendom

	Find relevant information for følgende:
	%s

	Betragt hvert af punkterne isoleret, de har ikke noget med hinanden at gøre og skal analyseres separat. Hvert punkt kan optræde flere gange (fx adresse der deles af virksomhed og person), medtag i de tilfælde alle matches.`, entityList)

	// Create HTTP client
	client := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Prepare the request payload using the new Responses API format
	requestBody := map[string]interface{}{
		"model": "gpt-4o-mini", // 200k tokens per minut limit (should be enough for 1000 pages)
		"input": []map[string]interface{}{
			{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type":     "input_file",
						"file_url": pdfURL,
					},
					{
						"type": "input_text",
						"text": userPrompt,
					},
				},
			},
		},
	}

	// Convert to JSON
	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return ExtractionResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/responses", bytes.NewBuffer(jsonData))
	if err != nil {
		return ExtractionResponse{}, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Initial delay
	delay := 1 * time.Second
	maxRetries := 3

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Printf("Attempt %d/%d, waiting %v before retry...", attempt+1, maxRetries, delay)
			time.Sleep(delay)
		}

		// Make the request
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("HTTP request error (attempt %d): %v", attempt+1, err)
			if attempt < maxRetries-1 {
				delay = delay * 2
				if delay > 60*time.Second {
					delay = 60 * time.Second
				}
				continue
			}
			return ExtractionResponse{}, fmt.Errorf("failed to make HTTP request: %w", err)
		}
		defer resp.Body.Close()

		// Read response
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return ExtractionResponse{}, fmt.Errorf("failed to read response body: %w", err)
		}

		// Check if request was successful
		if resp.StatusCode != http.StatusOK {
			log.Printf("OpenAI API error (attempt %d): HTTP %d - %s", attempt+1, resp.StatusCode, string(body))

			// Check if it's a rate limit error
			if resp.StatusCode == 429 {
				if attempt < maxRetries-1 {
					delay = delay * 2
					if delay > 60*time.Second {
						delay = 60 * time.Second
					}
					continue
				} else {
					return ExtractionResponse{}, fmt.Errorf("rate limit exceeded after %d retries", maxRetries)
				}
			} else {
				return ExtractionResponse{}, fmt.Errorf("OpenAI API error: HTTP %d - %s", resp.StatusCode, string(body))
			}
		}

		// Parse response
		var response map[string]interface{}
		if err := json.Unmarshal(body, &response); err != nil {
			return ExtractionResponse{}, fmt.Errorf("failed to parse response: %w", err)
		}

		// Check for API-level errors in the response
		if errorField, exists := response["error"]; exists && errorField != nil {
			return ExtractionResponse{}, fmt.Errorf("OpenAI API returned error: %v", errorField)
		}

		// Check if response is completed
		status, ok := response["status"].(string)
		if !ok || status != "completed" {
			return ExtractionResponse{}, fmt.Errorf("response not completed, status: %v", status)
		}

		// Extract the answer from the new Responses API format
		output, ok := response["output"].([]interface{})
		if !ok || len(output) == 0 {
			return ExtractionResponse{}, fmt.Errorf("no output in response")
		}

		outputItem, ok := output[0].(map[string]interface{})
		if !ok {
			return ExtractionResponse{}, fmt.Errorf("invalid output format")
		}

		content, ok := outputItem["content"].([]interface{})
		if !ok || len(content) == 0 {
			return ExtractionResponse{}, fmt.Errorf("no content in output")
		}

		contentItem, ok := content[0].(map[string]interface{})
		if !ok {
			return ExtractionResponse{}, fmt.Errorf("invalid content format")
		}

		answer, ok := contentItem["text"].(string)
		if !ok {
			return ExtractionResponse{}, fmt.Errorf("invalid text format")
		}

		log.Printf("Received answer, length: %d", len(answer))

		// Parse results - look for each entity in the response
		allResults := make(ExtractionResult)
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
		return ExtractionResponse{
			Results:     allResults,
			RawResponse: answer,
		}, nil
	}

	return ExtractionResponse{}, fmt.Errorf("failed to process document after %d attempts", maxRetries)
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
	panic("this should not be called")
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
