# Go Mail Styler

Go Mail Styler is a simple and flexible Go package for sending HTML emails via SMTP with support for template rendering, CSS inlining, attachments, and inline images.

---

## ✨ Features

- **Simple API** for sending HTML emails via SMTP
- **Two rendering modes**: file-based templates or direct HTML content strings
- **Template rendering** with dynamic data and Go template syntax
- **Built-in template functions** with support for custom functions
- **CSS inlining** - automatically convert external CSS to inline styles
- **Attachments & inline images** - embed files and images directly in emails
- **Performance optimized** - parallel CSS processing for faster rendering
- **Flexible initialization** - function options pattern for clean configuration
- **Lightweight** and easy to integrate

---

## 📦 Installation

```bash
go get github.com/phzeng0726/gomailstyler@v0.2.0
```

---

## 🚀 Quick Start

### Example 1: Using Template Files

```go
package main

import (
	"log"
	mailstyler "github.com/phzeng0726/gomailstyler"
)

func main() {
	// Initialize manager with options pattern
	manager, err := mailstyler.NewManagerWithOptions(
		mailstyler.WithSMTP("smtp.example.com", "587", "you@example.com"),
		mailstyler.WithTemplatePath("./templates"),
		mailstyler.WithCSSPath("./templates/css"),
	)
	if err != nil {
		log.Fatalf("failed to create manager: %v", err)
	}

	// Render template file with CSS inlining
	body, err := manager.RenderTemplateWithCSS("welcome.html", "styles.css", map[string]any{
		"Name": "John Doe",
	})
	if err != nil {
		log.Fatalf("failed to render template: %v", err)
	}

	// Send the email
	err = manager.SendMail(mailstyler.MailMessage{
		Subject: "Welcome to Our Service",
		Message: body,
		To:      []string{"recipient@example.com"},
	})
	if err != nil {
		log.Fatalf("failed to send mail: %v", err)
	}

	log.Println("Mail sent successfully!")
}
```

### Example 2: Using HTML Content Strings (New in v0.2.0)

```go
package main

import (
	"log"
	"html/template"
	mailstyler "github.com/phzeng0726/gomailstyler"
)

func main() {
	// Initialize manager (no template path needed for content mode)
	manager, err := mailstyler.NewManagerWithOptions(
		mailstyler.WithSMTP("smtp.example.com", "587", "you@example.com"),
	)
	if err != nil {
		log.Fatalf("failed to create manager: %v", err)
	}

	// Define HTML content as a string
	htmlContent := `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Hello {{.Name}}</title>
</head>
<body>
	<h1>Hello, {{toUpper .Name}}!</h1>
	<p>Your order #{{.OrderID}} has been confirmed.</p>
</body>
</html>
	`

	// Render HTML content with custom functions
	body, err := manager.RenderHTMLContentWithFuncs(htmlContent, map[string]any{
		"Name":    "Jane Smith",
		"OrderID": 12345,
	}, template.FuncMap{
		"toUpper": func(s string) string {
			return strings.ToUpper(s)
		},
	})
	if err != nil {
		log.Fatalf("failed to render content: %v", err)
	}

	// Send the email
	err = manager.SendMail(mailstyler.MailMessage{
		Subject: "Order Confirmation",
		Message: body,
		To:      []string{"recipient@example.com"},
	})
	if err != nil {
		log.Fatalf("failed to send mail: %v", err)
	}

	log.Println("Mail sent successfully!")
}
```

---

## 🏗️ Data Structures

### MailMessage

Represents an email message to be sent.

```go
type MailMessage struct {
	Subject      string          // Email subject line
	Message      string          // HTML body content
	To           []string        // Recipient email addresses
	Cc           []string        // CC email addresses (optional)
	Attachments  []Attachment    // File attachments (optional)
	InlineImages []InlineImage   // Inline images for HTML (optional)
}
```

### Attachment

Represents a file attachment.

```go
type Attachment struct {
	FileName string   // Attachment filename
	Data     []byte   // File binary data
}
```

### InlineImage

Represents an inline image embedded in HTML using `cid:` reference.

```go
type InlineImage struct {
	CID      string   // Content ID for HTML reference (e.g., "logo" for <img src="cid:logo">)
	FileName string   // Image filename
	Data     []byte   // Image binary data
}
```

**Example usage in HTML:**

```html
<img src="cid:logo" alt="Company Logo" />
```

---

## 📘 API Reference

### Initialization

#### `NewManagerWithOptions` (Recommended)

```go
func NewManagerWithOptions(opts ...ManagerOption) (*Manager, error)
```

Creates a new mail manager using the function options pattern.

**Available options:**

- `WithSMTP(server, port, sender string)` - **Required** - SMTP server configuration
- `WithTemplatePath(path string)` - Template directory (required for file-based rendering)
- `WithCSSPath(path string)` - CSS directory (required for CSS inlining)

**Example:**

```go
manager, err := mailstyler.NewManagerWithOptions(
	mailstyler.WithSMTP("smtp.example.com", "587", "sender@example.com"),
	mailstyler.WithTemplatePath("./templates"),      // Optional
	mailstyler.WithCSSPath("./templates/css"),       // Optional
)
```

---

### Sending Mail

#### `SendMail`

```go
func (m *Manager) SendMail(mm MailMessage) error
```

Sends an email with the provided message content.

**Parameters:**

- `mm` - A `MailMessage` struct containing subject, HTML body, recipients, and optional attachments/inline images

**MIME Structure:**

- Uses `multipart/mixed` for complex emails
- HTML body encoded as `text/html; charset="UTF-8"`
- Attachments use base64 encoding
- Inline images use `Content-ID` headers for HTML embedding

---

### File-Based Template Rendering

#### `RenderTemplate`

```go
func (m *Manager) RenderTemplate(tmplFile string, data any) (string, error)
```

Renders an HTML template file with dynamic data.

**Parameters:**

- `tmplFile` - Template filename (relative to template path)
- `data` - Data to inject into template

**Example:**

```go
body, err := manager.RenderTemplate("welcome.html", map[string]any{
	"Name": "John",
})
```

---

#### `RenderTemplateWithFuncs`

```go
func (m *Manager) RenderTemplateWithFuncs(tmplFile string, data any, customFuncs ...template.FuncMap) (string, error)
```

Renders template with custom template functions. Custom functions override default functions if names conflict.

**Example:**

```go
body, err := manager.RenderTemplateWithFuncs("invoice.html", data, template.FuncMap{
	"formatCurrency": func(amount float64) string {
		return fmt.Sprintf("$%.2f", amount)
	},
})
```

---

#### `RenderTemplateWithCSS`

```go
func (m *Manager) RenderTemplateWithCSS(tmplFile, cssFile string, data any) (string, error)
```

Renders template with CSS file and automatically inlines styles.

**Process:**

1. Template rendering and CSS file reading happen in parallel (performance optimization)
2. CSS inserted into `<head>` tag
3. CSS rules converted to inline `style` attributes
4. Class attributes removed after inlining

**Example:**

```go
body, err := manager.RenderTemplateWithCSS("newsletter.html", "styles.css", data)
```

---

#### `RenderTemplateWithFuncsAndCSS`

```go
func (m *Manager) RenderTemplateWithFuncsAndCSS(tmplFile, cssFile string, data any, customFuncs ...template.FuncMap) (string, error)
```

Combines custom functions and CSS inlining.

**Example:**

```go
body, err := manager.RenderTemplateWithFuncsAndCSS(
	"report.html",
	"report.css",
	data,
	template.FuncMap{
		"highlight": func(text string) string {
			return "<mark>" + text + "</mark>"
		},
	},
)
```

---

### HTML Content Rendering (New in v0.2.0)

#### `RenderHTMLContent`

```go
func (m *Manager) RenderHTMLContent(htmlContent string, data any) (string, error)
```

Renders HTML content directly from a string without file I/O.

**Use cases:**

- Dynamic email generation
- Simple emails without template files
- Programmatic content creation

**Example:**

```go
htmlContent := `<h1>Hello {{.Name}}!</h1><p>Thanks for signing up.</p>`
body, err := manager.RenderHTMLContent(htmlContent, map[string]any{
	"Name": "Alice",
})
```

---

#### `RenderHTMLContentWithFuncs`

```go
func (m *Manager) RenderHTMLContentWithFuncs(htmlContent string, data any, customFuncs ...template.FuncMap) (string, error)
```

Renders HTML content with custom template functions.

**Example:**

```go
htmlContent := `<h1>Order Total: {{formatPrice .Total}}</h1>`
body, err := manager.RenderHTMLContentWithFuncs(htmlContent, data, template.FuncMap{
	"formatPrice": func(price float64) string {
		return fmt.Sprintf("$%.2f", price)
	},
})
```

---

### Choosing a Rendering Method

| Feature                 | File-Based Templates          | HTML Content Strings           |
| ----------------------- | ----------------------------- | ------------------------------ |
| **File I/O**            | Required                      | Not required                   |
| **Template Management** | Separate files                | Inline in code                 |
| **CSS Inlining**        | ✅ Supported                  | ❌ Not supported               |
| **Custom Functions**    | ✅ Supported                  | ✅ Supported                   |
| **Reusability**         | High (shared template files)  | Medium (code-based)            |
| **Best for**            | Complex emails, design teams  | Simple emails, dynamic content |
| **Setup**               | Requires `WithTemplatePath()` | No path configuration needed   |

---

### 🧰 Default Template Functions

All rendering methods include these built-in functions:

| Function     | Description                         | Example                                              |
| ------------ | ----------------------------------- | ---------------------------------------------------- |
| `add`        | Adds two integers                   | `{{ add 1 2 }}` → `3`                                |
| `toUpper`    | Converts to uppercase               | `{{ toUpper "hello" }}` → `HELLO`                    |
| `toLower`    | Converts to lowercase               | `{{ toLower "HELLO" }}` → `hello`                    |
| `trim`       | Removes leading/trailing whitespace | `{{ trim " text " }}` → `text`                       |
| `title`      | Title case                          | `{{ title "go mail styler" }}` → `Go Mail Styler`    |
| `formatDate` | Formats time.Time                   | `{{ formatDate .Time "2006-01-02" }}` → `2025-04-23` |
| `isEmpty`    | Checks if string is empty           | `{{ isEmpty "  " }}` → `true`                        |
| `safeHTML`   | Marks string as safe HTML           | `{{ safeHTML "<b>bold</b>" }}`                       |
| `inc`        | Increments integer by 1             | `{{ inc 5 }}` → `6`                                  |

**Note:** Custom functions override default functions if names conflict.

---

## 📂 Examples

The `examples/` directory contains fully working examples:

### `01-template-file/` - File-Based Template Examples

- **01-basic** - Simple template rendering
- **02-with-css** - CSS inlining demonstration
- **03-with-functions** - Custom template functions
- **04-full-featured** - All features combined (template + CSS + functions + attachments + inline images)

### `02-html-content/` - HTML Content Examples (New in v0.2.0)

- **01-basic** - Basic HTML string rendering
- **02-with-attachments** - Content rendering with custom functions, attachments, and inline images

### Running Examples

```bash
cd examples/01-template-file/01-basic
# Create .env file with SMTP configuration
go run main.go
```

**Required .env variables:**

```env
SMTP_SERVER=smtp.example.com
SMTP_PORT=587
SMTP_SENDER=your-email@example.com
MAIL_RECEIVER=recipient@example.com
```

---

## ⚡ Performance Optimizations

### Parallel CSS Processing

When using `RenderTemplateWithCSS` or `RenderTemplateWithFuncsAndCSS`:

- Template rendering and CSS file reading execute **in parallel** using goroutines
- Results synchronized via channels for optimal performance
- Reduces total rendering time compared to sequential processing

### CSS Inlining Configuration

Uses [go-premailer](https://github.com/vanng822/go-premailer) with optimized settings:

- `KeepBangImportant: true` - Preserves `!important` declarations
- `RemoveClasses: true` - Removes class attributes after inlining (smaller HTML size)

---

## 🔧 Backward Compatibility

### Legacy Initialization (Deprecated but Supported)

```go
func NewManager(smtpServer, smtpPort, smtpSender, templatePath, cssPath string) (*Manager, error)
```

**Note:** This method is deprecated in favor of `NewManagerWithOptions` but remains available for backward compatibility.

**Migration example:**

```go
// Old way (still works)
manager, err := mailstyler.NewManager("smtp.example.com", "587", "sender@example.com", "./templates", "./css")

// New way (recommended)
manager, err := mailstyler.NewManagerWithOptions(
	mailstyler.WithSMTP("smtp.example.com", "587", "sender@example.com"),
	mailstyler.WithTemplatePath("./templates"),
	mailstyler.WithCSSPath("./css"),
)
```

---

## 📝 Example Template File

**`templates/welcome.html`:**

```html
<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <title>Welcome {{.Name}}</title>
  </head>
  <body>
    <h1>Hello, {{title .Name}}!</h1>
    <p>
      Welcome to our service. Your account was created on {{formatDate
      .SignupDate "January 2, 2006"}}.
    </p>
    <p>You have {{add .Credits 100}} bonus credits!</p>
  </body>
</html>
```

**Corresponding CSS file (`templates/css/styles.css`):**

```css
body {
  font-family: Arial, sans-serif;
  background-color: #f4f4f4;
  margin: 0;
  padding: 20px;
}

h1 {
  color: #333;
}

p {
  color: #666;
  line-height: 1.6;
}
```

---

## 🪪 License

[MIT](./LICENSE)

---

## 📚 Additional Resources

- **Repository:** [github.com/phzeng0726/gomailstyler](https://github.com/phzeng0726/gomailstyler)
- **Issues:** [Report bugs or request features](https://github.com/phzeng0726/gomailstyler/issues)
- **Examples:** See `examples/` directory for working code samples

---

**Made with ❤️ for Go developers who need reliable email delivery**
