# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**gomailstyler** is a Go package for sending HTML emails via SMTP with support for template rendering, CSS inlining, attachments, and inline images.

## Common Commands

### Build & Test

```bash
# Run the example application
cd examples
go run main.go

# Build the package
go build ./...

# Run go modules tidy
go mod tidy

# Format code
go fmt ./...
```

### Unit Tests

Follow [TESTING_GUIDELINES.md](TESTING_GUIDELINES.md) when writing tests (testify suite, white-box, fixtures in `testdata/`).

```bash
# Run all unit tests (no network, .env, or real SMTP server needed)
go test ./...

# Coverage (each package must stay >= 85%)
go test -cover ./...
```

### Manual Test with a Real SMTP Server

```bash
# Test with the example (requires .env file in examples/)
cd examples
# Create .env with: SMTP_HOST, SMTP_PORT, SMTP_SENDER, MAIL_RECEIVER
go run main.go
```

## Code Architecture

### Package Structure

The codebase follows a layered architecture:

```
gomailstyler/
├── manager.go          # Public API & Manager implementation
├── message.go          # Public message types (MailMessage, Attachment, InlineImage)
└── internal/service/   # Internal service layer
    ├── service.go      # Service initialization & dependency injection
    ├── template.go     # Template rendering (TemplatesService)
    └── css_tool.go     # CSS processing & inlining (CSSToolsService)
```

### Key Components

**Manager (manager.go)**

- Entry point for the library
- Implements the `MailManager` interface
- Delegates template rendering to internal services
- Handles SMTP mail sending with MIME multipart messages
- Builds email messages with HTML body, attachments, and inline images

**Services Layer (internal/service/)**

- `TemplatesService`: Handles Go template rendering with default template functions (add, toUpper, toLower, trim, title, formatDate, isEmpty, safeHTML, inc)
- `CSSToolsService`: Renders templates with CSS and converts CSS to inline styles using go-premailer
- Both services are created via dependency injection in `NewServices()`

### Data Flow

1. **Template Rendering Flow**:

   - User calls Manager method (e.g., `RenderTemplateWithFuncsAndCSS`)
   - Manager delegates to appropriate service (Templates or CSSTools)
   - Service loads template from `templatePath`, parses with optional functions
   - For CSS variants: CSS file loaded in parallel, inserted into HTML `<head>`, then inlined using premailer

2. **Email Sending Flow**:
   - User constructs `MailMessage` with rendered body
   - Manager builds MIME multipart message via `buildHTMLMessage()`
   - HTML body, attachments, and inline images encoded as separate MIME parts
   - Message sent via `smtp.SendMail()` with no authentication (plain SMTP)

### Critical Implementation Details

**MIME Message Construction** (manager.go:92-136)

- Uses multipart/mixed boundary for complex emails
- HTML body encoded as `text/html; charset="UTF-8"`
- Attachments use base64 encoding with 76-character line splits
- Inline images use `Content-ID` headers for HTML embedding (e.g., `cid:my-image`)
- Headers include `From`, `To`, `Subject`, optional `Cc`
- Recent fix: Removed duplicate `From` header in HTML section to pass email validation

**CSS Inlining** (css_tool.go:86-145)

- Template rendering and CSS file reading happen in parallel (goroutines + channels)
- CSS inserted into `<head>` using goquery DOM manipulation
- go-premailer converts `<style>` tags to inline styles with options:
  - `KeepBangImportant: true` - preserves `!important` declarations
  - `RemoveClasses: true` - strips class attributes after inlining

**Template Functions** (template.go:48-80)

- Default functions merged with user-provided custom functions
- Custom functions override defaults if name collision occurs
- All render methods log execution time
- Only the first `FuncMap` in `customFuncs` is used; later ones are ignored

**Known Issues** (pinned by `// KNOWN ISSUE:` tests in `manager_test.go`; update those tests when fixing)

- Header values are not checked for CR/LF, so `Subject`/`To`/`Cc` can inject extra headers
- `Subject` is written as raw UTF-8, not RFC 2047 encoded
- Headers are written from a map, so their order is random
- No `From`, `Date`, or `Message-ID` header is emitted
- `SendMail`'s `append(mm.To, mm.Cc...)` can write into the caller's `To` backing array
- HTML part has no `Content-Transfer-Encoding`; lines over 998 chars are possible

### SMTP Behavior

- Supports optional SMTP authentication via `WithSMTPPassword` option
- If password is provided, uses SMTP PLAIN authentication (`smtp.PlainAuth`)
- If no password, sends mail without authentication (suitable for local relays)
- `smtpSender` must match the authenticated user on most SMTP servers
- Port 587 is typical for SMTP submission with STARTTLS

### Environment Configuration

The `examples/` directory expects a `.env` file with:

- `SMTP_HOST`: SMTP server hostname
- `SMTP_PORT`: SMTP port (typically "587")
- `SMTP_SENDER`: From address
- `SMTP_PASSWORD`: SMTP password (optional, for authentication)
- `MAIL_RECEIVER`: Test recipient address
