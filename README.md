# Egobot - PDF Entity Extraction

A Go application that extracts information about specific entities from Danish Statstidende PDFs using OpenAI's GPT-3.5-turbo with intelligent content filtering.

## Features

- **PDF Processing**: Extracts text from PDF files using `ledongthuc/pdf`
- **AI Analysis**: Uses OpenAI GPT-3.5-turbo to analyze Danish Statstidende documents
- **Entity Extraction**: Finds information about specific entities (names, CPR/CVR numbers, addresses, etc.)
- **Intelligent Filtering**: Aggressive sentence-level filtering to avoid token limits without truncation
- **Robust Matching**: Multiple strategies for finding entities with formatting variations
- **REST API**: Simple HTTP endpoint for file upload and entity extraction
- **Email Automation**: Automated email processing with IMAP/SMTP integration

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
     -F 'entities=["Benny Gotfred Schmidt","0605410146","Lægårdsvej 12A"]'
   ```

### For Automated Email Processing (Complete System)

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
export OPENAI_STUB=false  # Use real OpenAI API
export SCHEDULE_CRON="0 0 9 * * *"  # Daily at 9 AM
export ENTITIES_TO_TRACK='["Benny Gotfred Schmidt","0605410146","Lægårdsvej 12A"]'
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

## API Usage

**Endpoint**: `POST /extract`

**Form Data**:
- `file`: PDF file to analyze
- `entities`: JSON array of entities to search for

**Response**: JSON object mapping each entity to extracted information

**Example Response**:
```json
{
  "Benny Gotfred Schmidt": "Found in dødsbo section: Benny Gotfred Schmidt, CPR: 0605410146, Address: Lægårdsvej 12A, 8000 Aarhus C",
  "0605410146": "CPR number found in dødsbo announcement",
  "Lægårdsvej 12A": "Address found in dødsbo section for Benny Gotfred Schmidt"
}
```

## Technical Implementation

### **🔍 Entity Matching Strategies**

The system uses multiple robust matching strategies to find entities even with formatting variations:

1. **Direct substring match** (normalized)
2. **Multi-word entity matching** (for names like "Benny Gotfred Schmidt")
3. **Format variation handling** (CPR numbers with/without spaces)
4. **Partial address matching** (finding address components separately)

### **📄 Content Filtering Approach**

To avoid token limits while preserving all relevant information:

1. **Early termination**: If no entities found, return immediately without API calls
2. **Sentence-level filtering**: Extract only sentences containing target entities or business keywords
3. **Ultra-aggressive filtering**: If still too long, extract only sentences with direct entity matches
4. **No truncation**: All filtering is content-based, not arbitrary truncation

### **⚡ Performance Optimizations**

- **GPT-3.5-turbo**: Higher rate limits (90k TPM vs 30k TPM for GPT-4o)
- **Exponential backoff**: Smart retry logic for rate limit handling
- **JSON array parsing**: Proper parsing of environment variable arrays
- **Early termination**: Saves API costs when entities aren't found

## Troubleshooting

### **🔧 Common Issues**

**Token Limit Errors:**
- **Error**: "This model's maximum context length is 16385 tokens"
- **Solution**: The aggressive filtering should prevent this. If still occurring, check if the PDF is extremely large.

**Entity Not Found:**
- **Check**: Verify entities are in the PDF using the robust matching
- **Check**: Ensure JSON array format is correct: `["entity1","entity2"]`
- **Check**: Look for formatting variations (spaces, punctuation)

**Rate Limit Errors:**
- **Error**: "429 Too Many Requests"
- **Solution**: The system includes exponential backoff and retry logic

### **🔧 Email Issues**

**SMTP Authentication Errors:**
- **535 Error**: "Username and Password not accepted"
  - ✅ **Solution**: Use App Passwords, not your regular Gmail password

**No PDF Emails Found:**
- ✅ **Check**: Emails must be from the last 24 hours
- ✅ **Check**: PDFs must be actual attachments (not embedded)

## Project Structure

```
egobot/
├── cmd/
│   ├── egobot/main.go          # HTTP API server
│   └── processor/main.go       # Email processor CLI
├── internal/
│   ├── ai/
│   │   ├── extractor.go        # OpenAI integration with filtering
│   │   ├── stub_extractor.go   # Stubbed responses for testing
│   │   └── stub_extractor_test.go
│   ├── config/
│   │   ├── config.go           # Configuration with JSON array parsing
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
ENTITIES_TO_TRACK=["Benny Gotfred Schmidt","0605410146","Lægårdsvej 12A"]

# Optional
OPENAI_STUB=false
SCHEDULE_CRON=0 6 * * *
```

**Cron Job:**
- **Schedule**: Daily at 6:00 AM CET
- **Command**: `go run ./cmd/processor -once`
- **Cost**: ~1 minute per day = 30 hours/month (well within free tier)

## Dependencies

- `github.com/gin-gonic/gin` - HTTP server
- `github.com/ledongthuc/pdf` - PDF text extraction
- `github.com/sashabaranov/go-openai` - OpenAI API client
- `go.uber.org/fx` - Dependency injection
- `github.com/emersion/go-imap` - IMAP email client
- `github.com/jordan-wright/email` - SMTP email sending
- `github.com/robfig/cron/v3` - Scheduled job processing 