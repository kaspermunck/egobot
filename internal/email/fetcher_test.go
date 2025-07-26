package email

import (
	"strings"
	"testing"
)

func TestIsStatstidendeEmail(t *testing.T) {
	fetcher := &EmailFetcher{}

	tests := []struct {
		subject string
		expect  bool
	}{
		{"Dagens kundgørelse (PDF) fra Statstidende.dk", true},
		{"Statstidende PDF", true},
		{"PDF from Statstidende", true},
		{"Regular email", false},
		{"Newsletter", false},
		{"", false},
	}

	for _, test := range tests {
		result := fetcher.isStatstidendeEmail(test.subject)
		if result != test.expect {
			t.Errorf("isStatstidendeEmail(%q) = %v, expected %v", test.subject, result, test.expect)
		}
	}
}

func TestFindStatstidendePDFLinks(t *testing.T) {
	fetcher := &EmailFetcher{}

	// Sample email content from the RTF file
	emailContent := `
		Some text here
		https://statstidende.dk/api/publication/3093/pdf
		More text
		Another link: https://statstidende.dk/api/publication/1234/pdf
		Regular link: https://example.com
	`

	links := fetcher.findStatstidendePDFLinks(emailContent)

	expectedLinks := []string{
		"https://statstidende.dk/api/publication/3093/pdf",
		"https://statstidende.dk/api/publication/1234/pdf",
	}

	if len(links) != len(expectedLinks) {
		t.Errorf("Expected %d links, got %d", len(expectedLinks), len(links))
	}

	for i, expected := range expectedLinks {
		if i >= len(links) {
			t.Errorf("Missing expected link: %s", expected)
			continue
		}
		if links[i] != expected {
			t.Errorf("Expected link %s, got %s", expected, links[i])
		}
	}
}

func TestFindStatstidendePDFLinksNoMatches(t *testing.T) {
	fetcher := &EmailFetcher{}

	emailContent := `
		Some text here
		https://example.com
		More text
		No PDF links here
	`

	links := fetcher.findStatstidendePDFLinks(emailContent)

	if len(links) != 0 {
		t.Errorf("Expected 0 links, got %d: %v", len(links), links)
	}
}

func TestFindStatstidendePDFLinksEmptyContent(t *testing.T) {
	fetcher := &EmailFetcher{}

	links := fetcher.findStatstidendePDFLinks("")

	if len(links) != 0 {
		t.Errorf("Expected 0 links for empty content, got %d", len(links))
	}
}

func TestFindStatstidendePDFLinksMultipleMatches(t *testing.T) {
	fetcher := &EmailFetcher{}

	emailContent := `
		https://statstidende.dk/api/publication/1/pdf
		https://statstidende.dk/api/publication/2/pdf
		https://statstidende.dk/api/publication/3/pdf
	`

	links := fetcher.findStatstidendePDFLinks(emailContent)

	if len(links) != 3 {
		t.Errorf("Expected 3 links, got %d", len(links))
	}

	// Check that all links follow the expected pattern
	for _, link := range links {
		if !strings.Contains(link, "statstidende.dk/api/publication/") {
			t.Errorf("Link doesn't match expected pattern: %s", link)
		}
		if !strings.HasSuffix(link, "/pdf") {
			t.Errorf("Link doesn't end with /pdf: %s", link)
		}
	}
}

func TestDownloadPDFInvalidURL(t *testing.T) {
	fetcher := &EmailFetcher{}

	// Test with an invalid URL
	_, err := fetcher.downloadPDF("https://invalid-url-that-does-not-exist.com/pdf")
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}
}

func TestDownloadPDFInvalidScheme(t *testing.T) {
	fetcher := &EmailFetcher{}

	// Test with invalid scheme
	_, err := fetcher.downloadPDF("ftp://example.com/file.pdf")
	if err == nil {
		t.Error("Expected error for invalid scheme, got nil")
	}
}
