# gomailstyler Examples

This directory contains comprehensive examples demonstrating all features of the gomailstyler package.

## 📁 Directory Structure

```
examples/
├── 01-template-file/          # Examples using external template files
│   ├── 01-basic/              # Basic template rendering
│   ├── 02-with-css/           # Template with CSS inlining
│   ├── 03-with-functions/     # Template with custom functions
│   └── 04-full-featured/      # All features combined
├── 02-html-content/           # Examples using HTML content strings
│   ├── 01-basic/              # Basic HTML content rendering
│   └── 02-with-attachments/   # HTML content with attachments
├── assets/                    # Shared resources
│   ├── templates/             # HTML template files
│   │   ├── welcome.html
│   │   ├── with_functions.html
│   │   ├── with_inline_images.html
│   │   └── css/
│   │       └── styles.css
│   └── images/
│       └── my_doggy.jpg
├── utils/                     # Shared utility functions
│   └── helpers.go
└── .env                       # SMTP configuration (create this)
```

## 🚀 Getting Started

### 1. Setup Environment

Create a `.env` file in the `examples/` directory with your SMTP settings:

```env
SMTP_HOST=smtp.example.com
SMTP_PORT=587
SMTP_SENDER=your-email@example.com
MAIL_RECEIVER=recipient@example.com
```

### 2. Install Dependencies

```bash
cd examples
go mod tidy
```

### 3. Run Examples

Navigate to any example directory and run:

```bash
# Example: Run basic template example
cd 01-template-file/01-basic
go run main.go

# Example: Run HTML content with attachments
cd 02-html-content/02-with-attachments
go run main.go
```

## 📚 Example Categories

### 01-template-file - External Template Files

These examples demonstrate using external template and CSS files.

#### 01-basic

**What it demonstrates:**

- Basic template rendering from a file
- Passing data to templates
- Using `NewManagerWithOptions()` with template path

**Files used:**

- Template: `assets/templates/welcome.html`

**Run:**

```bash
cd 01-template-file/01-basic && go run main.go
```

---

#### 02-with-css

**What it demonstrates:**

- CSS inlining from external CSS file
- Automatic conversion of `<style>` tags to inline styles
- Using `RenderTemplateWithCSS()`

**Files used:**

- Template: `assets/templates/welcome.html`
- CSS: `assets/templates/css/styles.css`

**Run:**

```bash
cd 01-template-file/02-with-css && go run main.go
```

---

#### 03-with-functions

**What it demonstrates:**

- Custom template functions (beyond default functions)
- Defining and using custom `FuncMap`
- Using `RenderTemplateWithFuncs()`

**Custom functions:**

- `bold` - Wraps text in `<strong>` tags
- `repeat` - Repeats a string N times

**Files used:**

- Template: `assets/templates/with_functions.html`

**Run:**

```bash
cd 01-template-file/03-with-functions && go run main.go
```

---

#### 04-full-featured

**What it demonstrates:**

- Combining all features:
  - Custom template functions
  - CSS inlining
  - File attachments
  - Inline images (using CID references)
- Using `RenderTemplateWithFuncsAndCSS()`
- Complete MIME multipart email construction

**Files used:**

- Template: `assets/templates/with_inline_images.html`
- CSS: `assets/templates/css/styles.css`
- Image: `assets/images/my_doggy.jpg`

**Run:**

```bash
cd 01-template-file/04-full-featured && go run main.go
```

---

### 02-html-content - Direct HTML Content

These examples demonstrate rendering HTML content directly (no template files).

#### 01-basic

**What it demonstrates:**

- Rendering HTML content from a string
- No template file I/O required
- Using `RenderHTMLContent()`
- Inline CSS styles in HTML

**Use case:**

- Dynamic HTML generation
- Simple emails without template files
- Programmatic email content

**Run:**

```bash
cd 02-html-content/01-basic && go run main.go
```

---

#### 02-with-attachments

**What it demonstrates:**

- HTML content rendering with custom functions
- File attachments
- Inline images with Content-ID (CID) references
- Using `RenderHTMLContentWithFuncs()`

**Features:**

- Custom template function: `toUpper`
- Attachment: `my_doggy.jpg`
- Inline image: `cid:my-doggy-img`

**Run:**

```bash
cd 02-html-content/02-with-attachments && go run main.go
```

---

## 🔧 Utility Functions

The `utils/helpers.go` file provides shared utilities:

- **LoadConfig()** - Loads SMTP configuration from `.env`
- **FileToBytes()** - Reads files into byte slices for attachments

## 🎯 Choosing the Right Example

| Use Case                                           | Example to Use                      |
| -------------------------------------------------- | ----------------------------------- |
| Simple template email                              | 01-template-file/01-basic           |
| Template with styling                              | 01-template-file/02-with-css        |
| Custom template logic                              | 01-template-file/03-with-functions  |
| Full-featured email (images, attachments, styling) | 01-template-file/04-full-featured   |
| Dynamic HTML without files                         | 02-html-content/01-basic            |
| Dynamic HTML with media                            | 02-html-content/02-with-attachments |

## 📖 Default Template Functions

All examples have access to these built-in functions:

| Function     | Description       | Example                                     |
| ------------ | ----------------- | ------------------------------------------- |
| `add`        | Addition          | `{{ add 1 2 }}` → `3`                       |
| `toUpper`    | Uppercase         | `{{ toUpper "hello" }}` → `HELLO`           |
| `toLower`    | Lowercase         | `{{ toLower "HELLO" }}` → `hello`           |
| `trim`       | Trim whitespace   | `{{ trim " text " }}` → `text`              |
| `title`      | Title case        | `{{ title "hello world" }}` → `Hello World` |
| `formatDate` | Format time       | `{{ formatDate .Time "2006-01-02" }}`       |
| `isEmpty`    | Check if empty    | `{{ isEmpty "" }}` → `true`                 |
| `safeHTML`   | Mark as safe HTML | `{{ safeHTML "<b>bold</b>" }}`              |
| `inc`        | Increment by 1    | `{{ inc 5 }}` → `6`                         |

## 🔍 Key Differences

### Template Files vs HTML Content

| Aspect          | Template Files                  | HTML Content                 |
| --------------- | ------------------------------- | ---------------------------- |
| **File I/O**    | Reads from disk                 | No file operations           |
| **Reusability** | Easy to maintain separate files | Content in code              |
| **Flexibility** | Better for templated emails     | Better for dynamic emails    |
| **CSS Support** | External CSS file support       | Inline styles only           |
| **Setup**       | Requires `WithTemplatePath()`   | No path configuration needed |

## 💡 Tips

1. **Use Template Files** when:

   - You have a library of email templates
   - Designers need to edit HTML separately
   - You want to separate concerns (code vs content)

2. **Use HTML Content** when:

   - Emails are generated programmatically
   - Content varies significantly per email
   - You want to minimize file dependencies

3. **CSS Inlining**:

   - Required for most email clients
   - Gmail, Outlook strip `<style>` tags
   - Use `RenderTemplateWithCSS()` for external CSS

4. **Inline Images**:
   - Use `cid:` references in `<img src="cid:image-id">`
   - Match `CID` field in `InlineImage` struct
   - Reduces external HTTP requests

## 📝 Environment Variables

| Variable        | Description             | Example            |
| --------------- | ----------------------- | ------------------ |
| `SMTP_HOST`     | SMTP server hostname    | `smtp.gmail.com`   |
| `SMTP_PORT`     | SMTP port (usually 587) | `587`              |
| `SMTP_SENDER`   | Sender email address    | `you@example.com`  |
| `MAIL_RECEIVER` | Test recipient email    | `test@example.com` |

## ⚠️ SMTP Notes

- Current implementation uses **plain SMTP** (no authentication)
- Suitable for:
  - Local mail servers
  - SMTP relays with IP-based auth
  - Authenticated SMTP (handled by relay layer)
- For production, consider adding authentication support

## 🤝 Contributing

When adding new examples:

1. Follow the numbering convention
2. Update this README with descriptions
3. Include clear comments in code
4. Test with actual SMTP server

## 📄 License

See the main project LICENSE file.
