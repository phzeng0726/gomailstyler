package main

import (
	"log"

	mailstyler "github.com/phzeng0726/gomailstyler"
	"github.com/phzeng0726/gomailstyler/examples/utils"
)

func main() {
	// Load configuration
	cfg := utils.LoadConfig()

	// Initialize manager with template and CSS paths
	// Configure SMTP password if provided
	opts := []mailstyler.ManagerOption{
		mailstyler.WithSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPSender),
		mailstyler.WithTemplatePath("../../assets/templates"),
		mailstyler.WithCSSPath("../../assets/templates/css"),
	}
	if cfg.SMTPPassword != "" {
		opts = append(opts, mailstyler.WithSMTPPassword(cfg.SMTPPassword))
	}

	manager, err := mailstyler.NewManagerWithOptions(opts...)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// Render template with CSS inlining
	body, err := manager.RenderTemplateWithCSS("welcome.html", "styles.css", map[string]any{
		"Name": "Charlie",
		"Age":  25,
	})
	if err != nil {
		log.Fatalf("Failed to render template with CSS: %v", err)
	}

	// Send email
	err = manager.SendMail(mailstyler.MailMessage{
		Subject: "Template with CSS Email",
		Message: body,
		To:      []string{cfg.MailReceiver},
	})
	if err != nil {
		log.Fatalf("Failed to send mail: %v", err)
	}

	log.Println("✓ Template with CSS email sent successfully!")
}
