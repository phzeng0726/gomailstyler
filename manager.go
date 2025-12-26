package mailstyler

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"html/template"
	"mime/multipart"
	"net/http"
	"net/smtp"
	"strings"

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
	smtpServer string
	smtpPort   string
	smtpSender string
	tmplSvc    service.Templates
	cssToolSvc service.CSSTools
}

// ManagerConfig holds configuration for the Manager
type ManagerConfig struct {
	smtpServer   string
	smtpPort     string
	smtpSender   string
	templatePath string
	cssPath      string
}

// validate checks if required fields are set
func (c *ManagerConfig) validate() error {
	if c.smtpServer == "" {
		return errors.New("empty smtpServer")
	}
	if c.smtpPort == "" {
		return errors.New("empty smtpPort")
	}
	if c.smtpSender == "" {
		return errors.New("empty smtpSender")
	}
	return nil
}

// ManagerOption is a functional option for configuring Manager
type ManagerOption func(*ManagerConfig)

// WithSMTP configures SMTP settings (required)
func WithSMTP(server, port, sender string) ManagerOption {
	return func(c *ManagerConfig) {
		c.smtpServer = server
		c.smtpPort = port
		c.smtpSender = sender
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

// NewManagerWithOptions creates a Manager using functional options pattern
func NewManagerWithOptions(opts ...ManagerOption) (*Manager, error) {
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
		smtpServer: cfg.smtpServer,
		smtpPort:   cfg.smtpPort,
		smtpSender: cfg.smtpSender,
		tmplSvc:    svc.Templates,
		cssToolSvc: svc.CSSTools,
	}, nil
}

// NewManager creates a Manager with SMTP and template configuration.
// Deprecated: Use NewManagerWithOptions for more flexible configuration.
func NewManager(smtpServer, smtpPort, smtpSender, templatePath, cssPath string) (*Manager, error) {
	return NewManagerWithOptions(
		WithSMTP(smtpServer, smtpPort, smtpSender),
		WithTemplatePath(templatePath),
		WithCSSPath(cssPath),
	)
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

func (m *Manager) buildHTMLMessage(mm MailMessage) []byte {
	buf := bytes.NewBuffer(nil)
	writer := multipart.NewWriter(buf)
	boundary := writer.Boundary()

	headers := map[string]string{
		"MIME-Version": "1.0",
		"From":         m.smtpSender,
		"To":           strings.Join(mm.To, ","),
		"Subject":      mm.Subject,
		"Content-Type": fmt.Sprintf("multipart/mixed; boundary=%s\n", boundary),
	}

	if len(mm.Cc) > 0 {
		headers["Cc"] = strings.Join(mm.Cc, ",")
	}

	for k, v := range headers {
		buf.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
	}

	// Write HTML message body
	buf.WriteString(fmt.Sprintf("--%s\r\n", boundary))
	buf.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	buf.WriteString("\r\n" + mm.Message)

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

	return buf.Bytes()
}

func (m *Manager) SendMail(mm MailMessage) error {
	addr := fmt.Sprintf("%s:%s", m.smtpServer, m.smtpPort)

	// Build HTML message
	msg := m.buildHTMLMessage(mm)

	// Sending email.
	allRecipients := append(mm.To, mm.Cc...)
	if err := smtp.SendMail(addr, nil, m.smtpSender, allRecipients, msg); err != nil {
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
