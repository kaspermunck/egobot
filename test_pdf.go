package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"egobot/internal/pdf"
)

func main() {
	// Open the sample PDF
	file, err := os.Open("statstidende_sample.pdf")
	if err != nil {
		log.Fatal("Failed to open PDF:", err)
	}
	defer file.Close()

	// Extract text
	text, err := pdf.ExtractText(file)
	if err != nil {
		log.Fatal("Failed to extract text:", err)
	}

	fmt.Printf("PDF length: %d characters\n", len(text))

	// Test with entities that ARE in the PDF
	testEntities := []string{"Jette Fries Lundsted", "0801620450", "Husmandsvej 1", "S17072025-152"}

	for _, entity := range testEntities {
		if idx := findKeyword(text, entity); idx != -1 {
			fmt.Printf("\nFound '%s' at position %d\n", entity, idx)
			start := max(0, idx-100)
			end := min(len(text), idx+200)
			fmt.Printf("Context: %s\n", text[start:end])
		} else {
			fmt.Printf("\nEntity '%s' NOT FOUND in PDF\n", entity)
		}
	}

	// Test the section extraction with actual content
	fmt.Printf("\n=== Testing section extraction with actual content ===\n")
	sections := []string{"Dødsboer", "Gældssanering", "Konkursboer", "Tvangsauktioner"}
	for _, section := range sections {
		if idx := strings.Index(text, section); idx != -1 {
			fmt.Printf("Found section '%s' at position %d\n", section, idx)
			// Show more of the section content
			start := idx
			end := min(len(text), idx+1000)
			fmt.Printf("Section content: %s\n", text[start:end])
		} else {
			fmt.Printf("Section '%s' NOT FOUND\n", section)
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func findKeyword(text, keyword string) int {
	return strings.Index(strings.ToLower(text), strings.ToLower(keyword))
}
