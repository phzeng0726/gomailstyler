package main

import (
	"io"
	"log"
	"os"

	"github.com/joho/godotenv"
	mailstyler "github.com/phzeng0726/gomailstyler"
)

var (
	cfg     *Config
	manager *mailstyler.Manager
)

// Config holds the environment variables for the mailer
type Config struct {
	SMTPServer   string
	SMTPPort     string
	SMTPSender   string
	MailReceiver string
}

// loadConfig loads environment variables into a Config struct
func loadConfig() *Config {
	err := godotenv.Load("../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	return &Config{
		SMTPServer:   os.Getenv("SMTP_SERVER"),   // e.g., "smtp.example.com"
		SMTPPort:     os.Getenv("SMTP_PORT"),     // e.g., "587"
		SMTPSender:   os.Getenv("SMTP_SENDER"),   // e.g., "you@example.com"
		MailReceiver: os.Getenv("MAIL_RECEIVER"), // e.g., "recipient@example.com"
	}
}

func fileToBytes(filePath string) ([]byte, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return io.ReadAll(file)
}

// Examples ==============================================
func sendContentMail(manager *mailstyler.Manager, receiver string) {
	htmlContent := `
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
</head>
<body style="font-family: Arial, sans-serif; padding: 20px;">
    <h1 style="color: #333;">Welcome, {{ .Name }}!</h1>
    <p style="font-size: 16px;">Your email is: {{ .Email }}</p>
</body>
</html>
`

	data := map[string]any{
		"Name":  "Alice",
		"Email": "alice@example.com",
	}

	body, err := manager.RenderHTMLContent(htmlContent, data)
	if err != nil {
		log.Fatalf("failed to render template: %v", err)
	}

	err = manager.SendMail(mailstyler.MailMessage{
		Subject: "Rendered HTML Content Template Email",
		Message: body,
		To:      []string{receiver},
	})
	if err != nil {
		log.Fatalf("failed to send mail: %v", err)
	}
}

func sendContentAttachmentsMail(manager *mailstyler.Manager, receiver string) {
	htmlContent := `
<!DOCTYPE html>
<html>
<head>
	<meta charset="UTF-8">
	<title>Welcome</title>
</head>
<body style="font-family: Arial, sans-serif; padding: 20px;">
	<h1 style="color: #333;">Welcome, {{ .Name }}!</h1>
	<p style="font-size: 16px;">Your email is: {{ .Email }}</p>

	<p>Here is my doggy:</p>
	<img src="cid:my-doggy-img" alt="My Doggy" style="max-width: 300px; border-radius: 8px;" />
</body>
</html>
`

	data := map[string]any{
		"Name":  "Alice",
		"Email": "alice@example.com",
	}

	// Render HTML content
	body, err := manager.RenderHTMLContent(htmlContent, data)
	if err != nil {
		log.Fatalf("failed to render HTML content: %v", err)
	}

	// Load attachment / inline image
	imageData, err := fileToBytes("../assets/images/my_doggy.jpg")
	if err != nil {
		log.Fatalf("failed to load image: %v", err)
	}

	// Send mail with attachment + inline image
	err = manager.SendMail(mailstyler.MailMessage{
		Subject: "HTML Content Email With Attachments",
		Message: body,
		To:      []string{receiver},
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
		log.Fatalf("failed to send mail: %v", err)
	}

	log.Println("HTML content mail with attachments sent successfully!")
}

// Run ==============================================
func init() {
	// Load configuration
	cfg = loadConfig()

	// Initialize mailer manager
	var err error
	manager, err = mailstyler.NewManagerWithOptions(
		mailstyler.WithSMTP(
			cfg.SMTPServer,
			cfg.SMTPPort,
			cfg.SMTPSender,
		),
	)
	if err != nil {
		log.Fatalf("failed to create manager: %v", err)
	}

}

func main() {
	// sendContentMail(manager, cfg.MailReceiver)
	sendContentAttachmentsMail(manager, cfg.MailReceiver)
}
