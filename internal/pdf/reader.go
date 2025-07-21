// Package pdf provides utilities for reading and extracting text from PDF files.
package pdf

import (
	"io"
	"os"
	"strings"

	"github.com/ledongthuc/pdf"
)

// ExtractText extracts all text from a PDF file reader.
func ExtractText(r io.Reader) (string, error) {
	tmpFile, err := os.CreateTemp("", "egobot_pdf_*.pdf")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmpFile.Name())
	_, err = io.Copy(tmpFile, r)
	if err != nil {
		tmpFile.Close()
		return "", err
	}
	tmpFile.Close()

	file, reader, err := pdf.Open(tmpFile.Name())
	if err != nil {
		return "", err
	}
	defer file.Close()

	var sb strings.Builder
	n := reader.NumPage()
	for i := 1; i <= n; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		content, _ := page.GetPlainText(nil)
		sb.WriteString(content)
	}
	return sb.String(), nil
}
