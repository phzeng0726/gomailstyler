package main

import (
	"log"
	"strings"

	mailstyler "github.com/phzeng0726/gomailstyler"
	"github.com/phzeng0726/gomailstyler/examples/utils"
)

const bccTemplate = `
<div style="font-family: Arial, sans-serif; max-width: 600px; padding: 20px;">
    <h2 style="color: #2c3e50;">{{ .Title }}</h2>
    <p>{{ .Body }}</p>
    <p style="color: #888;">Check the To / Cc fields: your own address should not appear anywhere.</p>
</div>
`

func main() {
	// Load configuration
	cfg := utils.LoadConfig()

	opts := []mailstyler.ManagerOption{
		mailstyler.WithSMTP(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPSender),
	}
	if cfg.SMTPPassword != "" {
		opts = append(opts, mailstyler.WithSMTPPassword(cfg.SMTPPassword))
	}

	manager, err := mailstyler.NewManager(opts...)
	if err != nil {
		log.Fatalf("Failed to create manager: %v", err)
	}

	// Use two different mailboxes for SMTP_SENDER and MAIL_RECEIVER: mail 1 is addressed To the sender,
	// so if both are the same, the receiver will see its own address in To.
	if strings.EqualFold(cfg.SMTPSender, cfg.MailReceiver) {
		log.Printf("⚠ SMTP_SENDER and MAIL_RECEIVER are the same (%s); mail 1 will show your address in To", cfg.MailReceiver)
	}

	// 1. To + Bcc: the visible recipient is the sender, the real recipient is hidden in Bcc.
	//    This is the recommended pattern; mail without any To is more likely to be flagged as spam.
	body, err := manager.RenderHTMLContent(bccTemplate, map[string]string{
		"Title": "To + Bcc",
		"Body":  "This mail is addressed to the sender. You received it as a Bcc recipient.",
	})
	if err != nil {
		log.Fatalf("Failed to render mail: %v", err)
	}

	err = manager.SendMail(mailstyler.MailMessage{
		Subject: "Bcc example 1/2: To + Bcc",
		Message: body,
		To:      []string{cfg.SMTPSender},
		Bcc:     []string{cfg.MailReceiver}, // only added to the SMTP envelope, never to the headers
	})
	if err != nil {
		log.Fatalf("Failed to send To + Bcc mail: %v", err)
	}
	log.Printf("✓ Sent To=%s, Bcc=%s", cfg.SMTPSender, cfg.MailReceiver)

	// 2. Bcc only: no To header is written at all.
	//    Mail clients show an empty or "undisclosed recipients" To field.
	body, err = manager.RenderHTMLContent(bccTemplate, map[string]string{
		"Title": "Bcc only",
		"Body":  "This mail has no To or Cc recipients. You received it as a Bcc recipient.",
	})
	if err != nil {
		log.Fatalf("Failed to render mail: %v", err)
	}

	err = manager.SendMail(mailstyler.MailMessage{
		Subject: "Bcc example 2/2: Bcc only",
		Message: body,
		Bcc:     []string{cfg.MailReceiver},
	})
	if err != nil {
		log.Fatalf("Failed to send Bcc-only mail: %v", err)
	}
	log.Printf("✓ Sent Bcc only to %s", cfg.MailReceiver)
}
