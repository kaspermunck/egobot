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

### For Automated Email Processing (Phase 2 - Email Infrastructure)

1. **Set up email credentials**:
   ```bash
   export IMAP_USERNAME=your-email@gmail.com
   export IMAP_PASSWORD=your-app-password
   export SMTP_FROM=your-email@gmail.com
   export SMTP_TO=recipient@example.com
   export OPENAI_STUB=true  # Use stubbed responses for testing
   ```

2. **Email infrastructure components**:
   - ✅ **IMAP Email Fetcher**: Connects to email servers and finds PDF attachments
   - ✅ **SMTP Email Sender**: Sends formatted HTML emails with analysis results
   - ✅ **HTML Templates**: Beautiful email formatting with entity results
   - ✅ **Error Handling**: Comprehensive error notifications
   - 🔄 **Email Processor**: Coming in Phase 3 (scheduling and processing logic)

## API Usage

**Endpoint**: `POST /extract`

**Form Data**:
- `file`: PDF file to analyze
- `entities`: JSON array of entities to search for

**Response**: JSON object mapping each entity to extracted information

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
│   └── processor/main.go       # Email processor (coming soon)
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
│   └── pdf/reader.go           # PDF text extraction
├── go.mod                      # Dependencies
└── statstidende_sample.pdf     # Sample PDF file
```

## Dependencies

- `github.com/gin-gonic/gin` - HTTP server
- `github.com/ledongthuc/pdf` - PDF text extraction
- `github.com/sashabaranov/go-openai` - OpenAI API client
- `go.uber.org/fx` - Dependency injection
- `github.com/emersion/go-imap` - IMAP email client
- `github.com/jordan-wright/email` - SMTP email sending
- `github.com/robfig/cron/v3` - Scheduled job processing 