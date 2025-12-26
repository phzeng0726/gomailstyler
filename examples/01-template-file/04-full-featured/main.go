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

	// Initialize manager with all paths
	manager, err := mailstyler.NewManagerWithOptions(
		mailstyler.WithSMTP(cfg.SMTPServer, cfg.SMTPPort, cfg.SMTPSender),
		mailstyler.WithTemplatePath("../../assets/templates"),
		mailstyler.WithCSSPath("../../assets/templates/css"),
	)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// Load image for attachment and inline display
	imageData, err := utils.FileToBytes("../../assets/images/my_doggy.jpg")
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}

	// Define custom template functions
	customFuncs := template.FuncMap{
		"emphasize": func(s string) string {
			return strings.ToUpper(s) + "!"
		},
	}

	// Render template with functions, CSS, and data
	body, err := manager.RenderTemplateWithFuncsAndCSS(
		"with_inline_images.html",
		"styles.css",
		map[string]any{
			"Name": "Pipi",
		},
		customFuncs,
	)
	if err != nil {
		log.Fatalf("Failed to render template: %v", err)
	}

	// Send email with attachments and inline images
	err = manager.SendMail(mailstyler.MailMessage{
		Subject: "Full-Featured Email with Everything!",
		Message: body,
		To:      []string{cfg.MailReceiver},
		Cc:      []string{}, // Optional: add CC recipients
		Attachments: []mailstyler.Attachment{
			{
				FileName: "my_doggy.jpg",
				Data:     imageData,
			},
		},
		InlineImages: []mailstyler.InlineImage{
			{
				CID:      "my-doggy-img",
				FileName: "my_doggy.jpg",
				Data:     imageData,
			},
		},
	})
	if err != nil {
		log.Fatalf("Failed to send mail: %v", err)
	}

	log.Println("✓ Full-featured email sent successfully!")
	log.Println("  - Custom template functions")
	log.Println("  - CSS inlining")
	log.Println("  - File attachments")
	log.Println("  - Inline images")
}
