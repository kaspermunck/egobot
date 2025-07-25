package email

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/smtp"
	"strings"
	"time"

	"egobot/internal/ai"
)

// EmailSender handles SMTP email sending
type EmailSender struct {
	config *SenderConfig
}

// SenderConfig holds email sending configuration
type SenderConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
	To       string
}

// NewEmailSender creates a new email sender
func NewEmailSender(config *SenderConfig) *EmailSender {
	return &EmailSender{
		config: config,
	}
}

// SendAnalysisResults sends an email with PDF analysis results
func (s *EmailSender) SendAnalysisResults(results []AnalysisResult) error {
	if len(results) == 0 {
		log.Printf("No analysis results to send")
		return nil
	}

	subject := fmt.Sprintf("PDF Analysis Results - %s", time.Now().Format("2006-01-02"))

	// Generate HTML content
	htmlContent, err := s.generateHTMLContent(results)
	if err != nil {
		return fmt.Errorf("failed to generate HTML content: %w", err)
	}

	// Send email
	return s.sendEmail(subject, htmlContent)
}

// AnalysisResult represents the result of analyzing a PDF
type AnalysisResult struct {
	Filename     string
	EmailSubject string
	EmailFrom    string
	EmailDate    time.Time
	Entities     ai.ExtractionResult
	Error        string
}

// generateHTMLContent generates HTML email content
func (s *EmailSender) generateHTMLContent(results []AnalysisResult) (string, error) {
	const htmlTemplate = `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>PDF Analysis Results</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .header { background-color: #f0f0f0; padding: 15px; border-radius: 5px; }
        .result { margin: 20px 0; padding: 15px; border: 1px solid #ddd; border-radius: 5px; }
        .entity { margin: 10px 0; padding: 10px; background-color: #f9f9f9; border-radius: 3px; }
        .entity-name { font-weight: bold; color: #333; }
        .entity-info { margin-top: 5px; color: #666; }
        .error { color: #d32f2f; background-color: #ffebee; padding: 10px; border-radius: 3px; }
        .summary { background-color: #e8f5e8; padding: 10px; border-radius: 3px; margin-top: 10px; }
    </style>
</head>
<body>
    <div class="header">
        <h1>PDF Analysis Results</h1>
        <p>Generated on {{.Timestamp}}</p>
        <p>Processed {{.Count}} PDF{{if ne .Count 1}}s{{end}}</p>
    </div>

    {{range .Results}}
    <div class="result">
        <h3>{{.Filename}}</h3>
        <p><strong>Email:</strong> {{.EmailSubject}} (from {{.EmailFrom}} on {{.EmailDate.Format "2006-01-02 15:04"}})</p>
        
        {{if .Error}}
        <div class="error">
            <strong>Error:</strong> {{.Error}}
        </div>
        {{else}}
            {{range $entity, $info := .Entities}}
            <div class="entity">
                <div class="entity-name">{{$entity}}</div>
                <div class="entity-info">{{$info}}</div>
            </div>
            {{end}}
        {{end}}
    </div>
    {{end}}

    <div class="summary">
        <h3>Summary</h3>
        <p>Total PDFs processed: {{.Count}}</p>
        <p>Successful analyses: {{.SuccessCount}}</p>
        <p>Failed analyses: {{.ErrorCount}}</p>
    </div>
</body>
</html>`

	tmpl, err := template.New("email").Parse(htmlTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Calculate summary statistics
	successCount := 0
	errorCount := 0
	for _, result := range results {
		if result.Error != "" {
			errorCount++
		} else {
			successCount++
		}
	}

	data := struct {
		Timestamp    string
		Count        int
		Results      []AnalysisResult
		SuccessCount int
		ErrorCount   int
	}{
		Timestamp:    time.Now().Format("2006-01-02 15:04:05"),
		Count:        len(results),
		Results:      results,
		SuccessCount: successCount,
		ErrorCount:   errorCount,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// sendEmail sends an email via SMTP
func (s *EmailSender) sendEmail(subject, htmlContent string) error {
	log.Printf("Sending email to %s via %s:%d", s.config.To, s.config.Host, s.config.Port)

	// Create email headers
	headers := make(map[string]string)
	headers["From"] = s.config.From
	headers["To"] = s.config.To
	headers["Subject"] = subject
	headers["MIME-Version"] = "1.0"
	headers["Content-Type"] = "text/html; charset=UTF-8"

	// Build email message
	var message bytes.Buffer
	for key, value := range headers {
		message.WriteString(fmt.Sprintf("%s: %s\r\n", key, value))
	}
	message.WriteString("\r\n")
	message.WriteString(htmlContent)

	// Send email with better error handling
	auth := smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)

	log.Printf("Attempting SMTP connection to %s with username: %s", addr, s.config.Username)

	if err := smtp.SendMail(addr, auth, s.config.From, []string{s.config.To}, message.Bytes()); err != nil {
		// Provide more helpful error messages for common Gmail issues
		if strings.Contains(err.Error(), "535") {
			return fmt.Errorf("SMTP authentication failed (535). For Gmail, ensure you're using an App Password, not your regular password. Enable 2FA and generate an App Password at https://myaccount.google.com/apppasswords")
		}
		if strings.Contains(err.Error(), "530") {
			return fmt.Errorf("SMTP authentication failed (530). Check your username and password")
		}
		if strings.Contains(err.Error(), "550") {
			return fmt.Errorf("SMTP authentication failed (550). Check your 'From' email address")
		}
		return fmt.Errorf("failed to send email: %w", err)
	}

	log.Printf("Email sent successfully")
	return nil
}

// SendErrorNotification sends an error notification email
func (s *EmailSender) SendErrorNotification(errorMsg string) error {
	subject := "PDF Analysis Error"

	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>PDF Analysis Error</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .error { color: #d32f2f; background-color: #ffebee; padding: 15px; border-radius: 5px; }
    </style>
</head>
<body>
    <h1>PDF Analysis Error</h1>
    <div class="error">
        <strong>Error occurred at:</strong> %s<br>
        <strong>Error message:</strong> %s
    </div>
</body>
</html>`, time.Now().Format("2006-01-02 15:04:05"), errorMsg)

	return s.sendEmail(subject, htmlContent)
}
