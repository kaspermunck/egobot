package main

import (
	"log"
	"os"
	"time"

	"egobot/internal/email"
)

func main() {
	// Set up test configuration
	config := &email.SenderConfig{
		Host:     "smtp.gmail.com",
		Port:     587,
		Username: os.Getenv("SMTP_USERNAME"),
		Password: os.Getenv("SMTP_PASSWORD"),
		From:     os.Getenv("SMTP_FROM"),
		To:       os.Getenv("SMTP_TO"),
	}

	if config.Username == "" || config.Password == "" || config.From == "" || config.To == "" {
		log.Fatal("Please set SMTP_USERNAME, SMTP_PASSWORD, SMTP_FROM, and SMTP_TO environment variables")
	}

	log.Printf("Testing SMTP connection...")
	log.Printf("Host: %s:%d", config.Host, config.Port)
	log.Printf("Username: %s", config.Username)
	log.Printf("From: %s", config.From)
	log.Printf("To: %s", config.To)

	sender := email.NewEmailSender(config)

	// Test with a simple result
	results := []email.AnalysisResult{
		{
			Filename:     "test.pdf",
			EmailSubject: "Test Email",
			EmailFrom:    "test@example.com",
			EmailDate:    time.Now(),
			Entities: map[string]string{
				"test": "Test entity found",
			},
		},
	}

	err := sender.SendAnalysisResults(results)
	if err != nil {
		log.Fatalf("Failed to send email: %v", err)
	}

	log.Printf("Email sent successfully!")
}
