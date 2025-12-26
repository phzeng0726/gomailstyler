package main

import (
	"html/template"
	"log"
	"strings"

	mailstyler "github.com/phzeng0726/gomailstyler"
	"github.com/phzeng0726/gomailstyler/examples/utils"
)

func main() {
	// Load configuration
	cfg := utils.LoadConfig()

	// Initialize manager
	manager, err := mailstyler.NewManagerWithOptions(
		mailstyler.WithSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPSender),
		mailstyler.WithTemplatePath("../../assets/templates"),
	)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// Define custom template functions
	customFuncs := template.FuncMap{
		"bold": func(s string) string {
			return "<strong>" + s + "</strong>"
		},
		"repeat": func(s string, count int) string {
			return strings.Repeat(s, count)
		},
	}

	// Render template with custom functions
	body, err := manager.RenderTemplateWithFuncs("with_functions.html", map[string]any{
		"Name": "Bob",
	}, customFuncs)
	if err != nil {
		log.Fatalf("Failed to render template with functions: %v", err)
	}

	// Send email
	err = manager.SendMail(mailstyler.MailMessage{
		Subject: "Template with Custom Functions Email",
		Message: body,
		To:      []string{cfg.MailReceiver},
	})
	if err != nil {
		log.Fatalf("Failed to send mail: %v", err)
	}

	log.Println("✓ Template with custom functions email sent successfully!")
}
