package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	mailstyler "github.com/phzeng0726/gomailstyler"
	"github.com/phzeng0726/gomailstyler/examples/utils"
)

const noticeTemplate = `
<div style="font-family: Arial, sans-serif; max-width: 600px; padding: 20px;">
    <h2 style="color: #c0392b;">Scheduled Maintenance</h2>
    <p>The system will be unavailable from <strong>{{ .Start }}</strong> to <strong>{{ .End }}</strong>.</p>
    <p style="color: #888;">Notice #{{ .NoticeID }}</p>
</div>
`

const completedTemplate = `
<div style="font-family: Arial, sans-serif; max-width: 600px; padding: 20px;">
    <h2 style="color: #27ae60;">Maintenance Completed</h2>
    <p>Maintenance finished at <strong>{{ .Finished }}</strong>. All services are back online.</p>
    <p style="color: #888;">Notice #{{ .NoticeID }}</p>
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

	// Message-IDs must be globally unique. Use a per-run notice ID and the sender's domain,
	// so running this example twice starts a new thread instead of reusing old IDs.
	noticeID := time.Now().Unix()
	userID := 7
	domain := senderDomain(cfg.SMTPSender)
	noticeMessageID := fmt.Sprintf("maintenance.%d.%d@%s", noticeID, userID, domain)

	// Non-ASCII subjects are RFC 2047 encoded automatically
	subject := fmt.Sprintf("Scheduled maintenance #%d (系統維護通知)", noticeID)
	data := map[string]any{
		"NoticeID": noticeID,
		"Start":    "2026-11-20 22:00",
		"End":      "2026-11-21 02:00",
		"Finished": time.Now().Format("2006-01-02 15:04"),
	}

	// 1. Send the maintenance notice with our own Message-ID
	noticeBody, err := manager.RenderHTMLContent(noticeTemplate, data)
	if err != nil {
		log.Fatalf("Failed to render notice: %v", err)
	}

	err = manager.SendMail(mailstyler.MailMessage{
		Subject:   subject,
		Message:   noticeBody,
		To:        []string{cfg.MailReceiver},
		MessageID: noticeMessageID,
		Headers: map[string]string{
			"X-Notice-ID":  strconv.FormatInt(noticeID, 10),
			"Thread-Topic": subject, // helps Outlook group the conversation
		},
	})
	if err != nil {
		log.Fatalf("Failed to send notice: %v", err)
	}
	log.Printf("✓ Notice sent (Message-ID: <%s>)", noticeMessageID)

	// Give the first mail a distinct Date so clients order the thread correctly
	time.Sleep(2 * time.Second)

	// 2. Send the completion notice as a reply in the same thread
	completedBody, err := manager.RenderHTMLContent(completedTemplate, data)
	if err != nil {
		log.Fatalf("Failed to render completion notice: %v", err)
	}

	err = manager.SendMail(mailstyler.MailMessage{
		// Keep the base subject identical: Gmail only threads when it matches after "RE: "
		Subject:   "RE: " + subject,
		Message:   completedBody,
		To:        []string{cfg.MailReceiver},
		MessageID: fmt.Sprintf("maintenance-done.%d.%d@%s", noticeID, userID, domain),
		InReplyTo: noticeMessageID, // References defaults to [InReplyTo]
		Headers: map[string]string{
			"X-Notice-ID":  strconv.FormatInt(noticeID, 10),
			"Thread-Topic": subject,
		},
	})
	if err != nil {
		log.Fatalf("Failed to send completion notice: %v", err)
	}
	log.Println("✓ Completion notice sent as a reply — check that both mails appear in one thread")
}

// senderDomain returns the domain part of an email address, used for Message-IDs
func senderDomain(addr string) string {
	if i := strings.LastIndex(addr, "@"); i >= 0 && i < len(addr)-1 {
		return addr[i+1:]
	}
	return "localhost"
}
