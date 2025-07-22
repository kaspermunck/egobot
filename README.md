# Egobot - PDF Entity Extraction

A Go application that extracts information about specific entities (companies, VAT numbers, persons, industries) from Danish company registry PDFs using OpenAI's GPT-4o.

## Features

- **PDF Processing**: Extracts text from PDF files using `ledongthuc/pdf`
- **AI Analysis**: Uses OpenAI GPT-4o to analyze Danish company registry documents
- **Entity Extraction**: Finds information about specific entities (bankruptcy, acquisitions, management changes, etc.)
- **Large File Support**: Chunks large PDFs to avoid token limits
- **REST API**: Simple HTTP endpoint for file upload and entity extraction

## Quick Start

### For Manual Testing (Existing API)

1. **Set up OpenAI API key**:
   ```bash
   export OPENAI_API_KEY=sk-your-key-here
   ```

2. **Start the server**:
   ```bash
   go run ./cmd/egobot
   ```

3. **Test the API**:
   ```bash
   curl -X POST http://localhost:8080/extract \
     -F "file=@statstidende_sample.pdf" \
     -F 'entities=["Danske Bank","12345678","fintech","John Doe"]'
   ```

### For Automated Email Processing (Phase 3 - Complete System)

#### **📧 Email Configuration Setup**

**Step 1: Enable 2-Factor Authentication**
1. Go to [Google Account Security](https://myaccount.google.com/security)
2. Enable "2-Step Verification" (required for App Passwords)

**Step 2: Generate App Passwords**
1. Go to [App Passwords](https://myaccount.google.com/apppasswords)
2. Generate two app passwords:
   - **IMAP**: Select "Mail" → "Other (Custom name)" → Name it "egobot-imap"
   - **SMTP**: Select "Mail" → "Other (Custom name)" → Name it "egobot-smtp"
3. Copy both 16-character passwords

**Step 3: Set Environment Variables**
```bash
# IMAP Configuration (for fetching emails)
export IMAP_USERNAME=your-email@gmail.com
export IMAP_PASSWORD=your-16-char-imap-app-password

# SMTP Configuration (for sending results)
export SMTP_USERNAME=your-email@gmail.com
export SMTP_PASSWORD=your-16-char-smtp-app-password
export SMTP_FROM=your-email@gmail.com
export SMTP_TO=recipient@example.com

# Processing Configuration
export OPENAI_STUB=true  # Use stubbed responses for testing
export SCHEDULE_CRON="0 0 9 * * *"  # Daily at 9 AM
export ENTITIES_TO_TRACK='["Danske Bank", "fintech", "bankruptcy"]'
```

**Step 4: Test Email Configuration**
```bash
# Test SMTP connection
go run test_smtp.go

# Test complete system
go run ./cmd/processor -once
```

#### **🚀 Running the Email Processor**

```bash
# Run once immediately
go run ./cmd/processor -once

# Show schedule information
go run ./cmd/processor -schedule

# Start scheduled processing (runs continuously)
go run ./cmd/processor
```

#### **📋 Complete System Components**
- ✅ **IMAP Email Fetcher**: Connects to Gmail and finds PDF attachments
- ✅ **SMTP Email Sender**: Sends beautiful HTML emails with analysis results
- ✅ **HTML Templates**: Professional email formatting with entity results
- ✅ **Error Handling**: Enhanced error messages for Gmail authentication
- ✅ **Email Processor**: Orchestrates fetching, analysis, and sending
- ✅ **Cron Scheduler**: Automated scheduling with retry logic
- ✅ **Command Line Tool**: `cmd/processor` with multiple modes

## API Usage

**Endpoint**: `POST /extract`

**Form Data**:
- `file`: PDF file to analyze
- `entities`: JSON array of entities to search for

**Response**: JSON object mapping each entity to extracted information

## Troubleshooting

### **🔧 Common Email Issues**

**SMTP Authentication Errors:**
- **535 Error**: "Username and Password not accepted"
  - ✅ **Solution**: Use App Passwords, not your regular Gmail password
  - ✅ **Steps**: Enable 2FA → Generate App Password → Use 16-char password

**IMAP Connection Issues:**
- **Connection refused**: Check if IMAP is enabled in Gmail settings
- **Authentication failed**: Use App Password for IMAP as well

**No PDF Emails Found:**
- ✅ **Check**: Emails must be from the last 24 hours
- ✅ **Check**: PDFs must be actual attachments (not embedded)
- ✅ **Check**: IMAP folder setting (default: "INBOX")

**Testing Commands:**
```bash
# Test SMTP only
go run test_smtp.go

# Test IMAP only (check logs for connection details)
export IMAP_USERNAME=your-email@gmail.com
export IMAP_PASSWORD=your-app-password
go run ./cmd/processor -once
```

## OpenAI API Learnings

### Token Limits & Rate Limits
- **Token Limit**: GPT-4o has ~30k tokens per request limit
- **Rate Limit**: 30k tokens per minute (TPM) for organization
- **Solution**: Chunk large PDFs into ~2000 token pieces

### API Reliability Issues
- **Assistants API**: File attachment fields (`file_ids`) are broken in `go-openai` library
- **Chat Completions**: More reliable for text analysis
- **File Upload**: Works for file upload but not for direct attachment to messages

### Best Practices
- Extract text from PDFs first, then send to AI
- Use chunking for large documents
- Implement comprehensive error logging
- Handle rate limits gracefully

## Project Structure

```
egobot/
├── cmd/
│   ├── egobot/main.go          # HTTP API server
│   └── processor/main.go       # Email processor CLI
├── internal/
│   ├── ai/
│   │   ├── extractor.go        # OpenAI integration
│   │   ├── stub_extractor.go   # Stubbed responses for testing
│   │   └── stub_extractor_test.go
│   ├── config/
│   │   ├── config.go           # Configuration management
│   │   └── config_test.go
│   ├── email/
│   │   ├── fetcher.go          # IMAP email fetching
│   │   ├── sender.go           # SMTP email sending
│   │   ├── fetcher_test.go     # Email fetcher tests
│   │   └── sender_test.go      # Email sender tests
│   ├── processor/
│   │   ├── processor.go        # Email processing orchestration
│   │   └── processor_test.go   # Processor tests
│   ├── scheduler/
│   │   ├── scheduler.go        # Cron-based scheduling
│   │   └── scheduler_test.go   # Scheduler tests
│   └── pdf/reader.go           # PDF text extraction
├── go.mod                      # Dependencies
└── statstidende_sample.pdf     # Sample PDF file
```

## Production Deployment

### **🚀 Railway Deployment (Recommended)**

**Why Railway?**
- ✅ **Free tier**: 500 hours/month (perfect for daily cron)
- ✅ **Single command**: `./deploy.sh`
- ✅ **Cron support**: Built-in scheduled jobs
- ✅ **Environment variables**: Easy management
- ✅ **Git integration**: Automatic deployments

**Quick Deploy:**
```bash
# Install Railway CLI
npm install -g @railway/cli

# Deploy with one command
./deploy.sh
```

**Environment Variables to Set in Railway Dashboard:**
```bash
# Required
IMAP_USERNAME=your-email@gmail.com
IMAP_PASSWORD=your-app-password
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM=your-email@gmail.com
SMTP_TO=recipient@example.com
OPENAI_API_KEY=your-openai-key
ENTITIES_TO_TRACK=["Danske Bank", "fintech", "bankruptcy"]

# Optional
OPENAI_STUB=false
SCHEDULE_CRON=0 6 * * *
```

**Cron Job:**
- **Schedule**: Daily at 6:00 AM CET
- **Command**: `go run ./cmd/processor -once`
- **Cost**: ~1 minute per day = 30 hours/month (well within free tier)

### **🔧 Railway Troubleshooting**

**"No service could be found" error:**
```bash
# Link to your existing project
railway link

# Then deploy
./deploy.sh
```

**"Not logged in" error:**
```bash
railway login
```

**"Project not found" error:**
```bash
railway link
```

### **🔧 Alternative: Render**

If you prefer Render:
```bash
# Install Render CLI
npm install -g @render/cli

# Deploy
render deploy
```

## Dependencies

- `github.com/gin-gonic/gin` - HTTP server
- `github.com/ledongthuc/pdf` - PDF text extraction
- `github.com/sashabaranov/go-openai` - OpenAI API client
- `go.uber.org/fx` - Dependency injection
- `github.com/emersion/go-imap` - IMAP email client
- `github.com/jordan-wright/email` - SMTP email sending
- `github.com/robfig/cron/v3` - Scheduled job processing 