package main

import (
	"log"

	mailstyler "github.com/phzeng0726/gomailstyler"
	"github.com/phzeng0726/gomailstyler/examples/utils"
)

func main() {
	// Load configuration
	cfg := utils.LoadConfig()

	// Initialize manager with template path
	// Configure SMTP password if provided
	opts := []mailstyler.ManagerOption{
		mailstyler.WithSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPSender),
		mailstyler.WithTemplatePath("../../assets/templates"),
	}
	if cfg.SMTPPassword != "" {
		opts = append(opts, mailstyler.WithSMTPPassword(cfg.SMTPPassword))
	}

	manager, err := mailstyler.NewManagerWithOptions(opts...)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// Render template with data
	body, err := manager.RenderTemplate("welcome.html", map[string]any{
		"Name": "Alice",
		"Age":  30,
	})
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	// Send email
	err = manager.SendMail(mailstyler.MailMessage{
		Subject: "Basic Template Email",
		Message: body,
		To:      []string{cfg.MailReceiver},
	})
	if err != nil {
		log.Fatalf("Failed to send mail: %v", err)
	}

	log.Println("✓ Basic template email sent successfully!")
}
