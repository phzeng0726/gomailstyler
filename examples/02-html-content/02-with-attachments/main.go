package main

import (
	"html/template"
	"log"

	mailstyler "github.com/phzeng0726/gomailstyler"
	"github.com/phzeng0726/gomailstyler/examples/utils"
)

func main() {
	// Load configuration
	cfg := utils.LoadConfig()

	// Initialize manager
	manager, err := mailstyler.NewManagerWithOptions(
		mailstyler.WithSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPSender),
	)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// Define HTML content with inline image reference
	htmlContent := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <title>Welcome</title>
</head>
<body style="font-family: Arial, sans-serif; padding: 20px; background-color: #f5f5f5;">
    <div style="max-width: 600px; margin: 0 auto; background: white; padding: 30px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1);">
        <h1 style="color: #2c3e50; border-bottom: 3px solid #3498db; padding-bottom: 10px;">
            Hello, {{ toUpper .Name }}!
        </h1>
        <p style="font-size: 16px; color: #555; line-height: 1.6;">
            Your email is: <strong style="color: #3498db;">{{ .Email }}</strong>
        </p>

        <div style="margin: 20px 0;">
            <h2 style="color: #34495e; font-size: 18px;">Check out this image:</h2>
            <img src="cid:my-doggy-img" alt="My Doggy" style="max-width: 100%; border-radius: 8px; box-shadow: 0 4px 6px rgba(0,0,0,0.1);" />
        </div>

        <p style="color: #7f8c8d; font-size: 14px; margin-top: 30px; padding-top: 20px; border-top: 1px solid #ecf0f1;">
            This email includes an attachment and an inline image!
        </p>
    </div>
</body>
</html>
`

	// Define custom functions
	customFuncs := template.FuncMap{
		"toUpper": func(s string) string {
			return s
		},
	}

	// Render HTML content with custom functions
	body, err := manager.RenderHTMLContentWithFuncs(htmlContent, map[string]any{
		"Name":  "Charlie",
		"Email": "charlie@example.com",
	}, customFuncs)
	if err != nil {
		log.Fatalf("Failed to render HTML content: %v", err)
	}

	// Load image data
	imageData, err := utils.FileToBytes("../../assets/images/my_doggy.jpg")
	if err != nil {
		log.Fatalf("Failed to load image: %v", err)
	}

	// Send email with attachment and inline image
	err = manager.SendMail(mailstyler.MailMessage{
		Subject: "HTML Content with Attachments",
		Message: body,
		To:      []string{cfg.MailReceiver},
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

	log.Println("✓ HTML content email with attachments sent successfully!")
	log.Println("  - Custom template functions")
	log.Println("  - File attachment")
	log.Println("  - Inline image")
}
