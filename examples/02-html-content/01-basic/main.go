package main

import (
	"log"

	mailstyler "github.com/phzeng0726/gomailstyler"
	"github.com/phzeng0726/gomailstyler/examples/utils"
)

func main() {
	// Load configuration
	cfg := utils.LoadConfig()

	// Initialize manager (no template path needed for HTML content)
	manager, err := mailstyler.NewManagerWithOptions(
		mailstyler.WithSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPSender),
	)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// Define HTML content directly
	htmlContent := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome</title>
</head>
<body style="font-family: Arial, sans-serif; padding: 20px; background-color: #f5f5f5;">
    <div style="max-width: 600px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px;">
        <h1 style="color: #2c3e50;">Hello, {{ .Name }}!</h1>
        <p style="font-size: 16px; color: #555; line-height: 1.6;">
            Welcome to our service. Your email is: <strong>{{ .Email }}</strong>
        </p>
        <p style="color: #888; font-size: 14px;">
            Age: {{ .Age }}
        </p>
    </div>
</body>
</html>
`

	// Render HTML content with data
	body, err := manager.RenderHTMLContent(htmlContent, map[string]any{
		"Name":  "Alice",
		"Email": "alice@example.com",
		"Age":   30,
	})
	if err != nil {
		log.Fatalf("Failed to render HTML content: %v", err)
	}

	// Send email
	err = manager.SendMail(mailstyler.MailMessage{
		Subject: "Basic HTML Content Email",
		Message: body,
		To:      []string{cfg.MailReceiver},
	})
	if err != nil {
		log.Fatalf("Failed to send mail: %v", err)
	}

	log.Println("✓ HTML content email sent successfully!")
}
