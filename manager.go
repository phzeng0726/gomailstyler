package mailstyler

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"mime/multipart"
	"mime/quotedprintable"
	"net/http"
	"net/smtp"
	"time"

	"github.com/phzeng0726/gomailstyler/internal/service"
)

type MailManager interface {
	SendMail(mm MailMessage) error

	// File-based rendering
	RenderTemplate(tmplFile string, data any) (string, error)
	RenderTemplateWithFuncs(tmplFile string, data any, customFuncs ...template.FuncMap) (string, error)
	RenderTemplateWithCSS(tmplFile, cssFile string, data any) (string, error)
	RenderTemplateWithFuncsAndCSS(tmplFile, cssFile string, data any, customFuncs ...template.FuncMap) (string, error)

	// Content-based rendering
	RenderHTMLContent(htmlContent string, data any) (string, error)
	RenderHTMLContentWithFuncs(htmlContent string, data any, customFuncs ...template.FuncMap) (string, error)
}

type Manager struct {
	smtpHost     string
	smtpPort     string
	smtpSender   string
	smtpPassword *string
	tmplSvc      service.Templates
	cssToolSvc   service.CSSTools
	now          func() time.Time // source of the Date header; replaced in tests
}

// ManagerConfig holds configuration for the Manager
type ManagerConfig struct {
	smtpHost     string
	smtpPort     string
	smtpSender   string
	smtpPassword *string
	templatePath string
	cssPath      string
}

// validate checks if required fields are set
func (c *ManagerConfig) validate() error {
	if c.smtpHost == "" {
		return errors.New("empty smtpHost")
	}
	if c.smtpPort == "" {
		return errors.New("empty smtpPort")
	}
	if c.smtpSender == "" {
		return errors.New("empty smtpSender")
	}
	// smtpSender is written into the From header
	if err := validateHeaderValue("smtpSender", c.smtpSender); err != nil {
		return err
	}
	return nil
}

// ManagerOption is a functional option for configuring Manager
type ManagerOption func(*ManagerConfig)

// WithSMTP configures SMTP settings (required)
func WithSMTP(server, port, sender string) ManagerOption {
	return func(c *ManagerConfig) {
		c.smtpHost = server
		c.smtpPort = port
		c.smtpSender = sender
	}
}

// WithSMTPPassword sets the SMTP password for authentication (optional)
// If not provided, the mail will be sent without authentication
func WithSMTPPassword(password string) ManagerOption {
	return func(c *ManagerConfig) {
		c.smtpPassword = &password
	}
}

// WithTemplatePath sets the root path for template files (optional)
// Required only if using file-based rendering methods
func WithTemplatePath(path string) ManagerOption {
	return func(c *ManagerConfig) {
		c.templatePath = path
	}
}

// WithCSSPath sets the root path for CSS files (optional)
// Required only if using CSS-based rendering methods
func WithCSSPath(path string) ManagerOption {
	return func(c *ManagerConfig) {
		c.cssPath = path
	}
}

// NewManager creates a Manager using functional options pattern
func NewManager(opts ...ManagerOption) (*Manager, error) {
	cfg := &ManagerConfig{}

	// Apply all options
	for _, opt := range opts {
		opt(cfg)
	}

	// Validate required fields
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid manager config: %w", err)
	}

	// Create services
	svc := service.NewServices(service.Deps{
		TmplPath: cfg.templatePath,
		CSSPath:  cfg.cssPath,
	})

	return &Manager{
		smtpHost:     cfg.smtpHost,
		smtpPort:     cfg.smtpPort,
		smtpSender:   cfg.smtpSender,
		smtpPassword: cfg.smtpPassword,
		tmplSvc:      svc.Templates,
		cssToolSvc:   svc.CSSTools,
		now:          time.Now,
	}, nil
}

func (m *Manager) writeHTMLAttachment(
	buf *bytes.Buffer,
	boundary string,
	data []byte,
	fileName, disposition string,
	contentID *string,
) {
	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	buf.WriteString(fmt.Sprintf("Content-Type: %s\r\n", http.DetectContentType(data)))
	buf.WriteString("Content-Transfer-Encoding: base64\r\n")
	buf.WriteString(fmt.Sprintf("Content-Disposition: %s; filename=\"%s\"\r\n", disposition, fileName))

	if contentID != nil {
		buf.WriteString(fmt.Sprintf("Content-ID: <%s>\r\n", *contentID))
	}

	buf.WriteString("\r\n")

	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(data)))
	base64.StdEncoding.Encode(encoded, data)

	// Split into 76-character lines
	for i := 0; i < len(encoded); i += 76 {
		end := i + 76
		if end > len(encoded) {
			end = len(encoded)
		}
		buf.Write(encoded[i:end])
		buf.WriteString("\r\n")
	}
}

func (m *Manager) buildHTMLMessage(mm MailMessage) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()

	// Validate and order all headers before writing anything
	headers, err := m.buildHeaders(mm, boundary)
	if err != nil {
		return nil, err
	}

	for _, h := range headers {
		fmt.Fprintf(buf, "%s: %s\r\n", h.name, h.value)
	}

	// Write HTML message body as quoted-printable to keep lines short and 7-bit
	buf.WriteString(fmt.Sprintf("\r\n--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString("Content-Transfer-Encoding: quoted-printable\r\n\r\n")

	qp := quotedprintable.NewWriter(buf)
	qp.Write([]byte(mm.Message))
	qp.Close()

	// Add attachments
	if len(mm.Attachments) > 0 {
		for _, attachment := range mm.Attachments {
			m.writeHTMLAttachment(buf, boundary, attachment.Data, attachment.FileName, "attachment", nil)
		}
	}

	// Add inline images
	if len(mm.InlineImages) > 0 {
		for _, img := range mm.InlineImages {
			m.writeHTMLAttachment(buf, boundary, img.Data, img.FileName, "inline", &img.CID)
		}
	}

	// Final boundary to indicate end of MIME message
	buf.WriteString(fmt.Sprintf("\r\n--%s--\r\n", boundary))

	return buf.Bytes(), nil
}

func (m *Manager) SendMail(mm MailMessage) error {
	// Build HTML message first so invalid input fails before connecting
	msg, err := m.buildHTMLMessage(mm)
	if err != nil {
		return err
	}

	addr := fmt.Sprintf("%s:%s", m.smtpHost, m.smtpPort)

	// Copy into a new slice so the caller's backing arrays are never modified.
	// Bcc only goes into the SMTP envelope, never into the message headers.
	allRecipients := make([]string, 0, len(mm.To)+len(mm.Cc)+len(mm.Bcc))
	allRecipients = append(allRecipients, mm.To...)
	allRecipients = append(allRecipients, mm.Cc...)
	allRecipients = append(allRecipients, mm.Bcc...)

	// Set up authentication if a password is provided
	var auth smtp.Auth
	if m.smtpPassword != nil && *m.smtpPassword != "" {
		auth = smtp.PlainAuth("", m.smtpSender, *m.smtpPassword, m.smtpHost)
	}

	// Sending email
	if err := smtp.SendMail(addr, auth, m.smtpSender, allRecipients, msg); err != nil {
		return err
	}

	return nil
}

func (m *Manager) RenderTemplate(tmplFile string, data any) (string, error) {
	return m.tmplSvc.RenderTemplate(tmplFile, data)
}

func (m *Manager) RenderTemplateWithFuncs(tmplFile string, data any, customFuncs ...template.FuncMap) (string, error) {
	return m.tmplSvc.RenderTemplateWithFuncs(tmplFile, data, customFuncs)
}

func (m *Manager) RenderTemplateWithCSS(tmplFile, cssFile string, data any) (string, error) {
	return m.cssToolSvc.RenderTemplateWithCSS(tmplFile, cssFile, data)
}

func (m *Manager) RenderTemplateWithFuncsAndCSS(tmplFile, cssFile string, data any, customFuncs ...template.FuncMap) (string, error) {
	return m.cssToolSvc.RenderTemplateWithFuncsAndCSS(tmplFile, cssFile, data, customFuncs)

}

func (m *Manager) RenderHTMLContent(content string, data any) (string, error) {
	return m.tmplSvc.RenderHTMLContent(content, data)
}

func (m *Manager) RenderHTMLContentWithFuncs(tmplFile string, data any, customFuncs ...template.FuncMap) (string, error) {
	return m.tmplSvc.RenderHTMLContentWithFuncs(tmplFile, data, customFuncs)
}
