package mailstyler

type MailMessage struct {
	Subject      string
	Message      string
	To           []string
	Cc           []string
	Attachments  []Attachment
	InlineImages []InlineImage

	// MessageID sets the Message-ID header, with or without angle brackets
	// (e.g. "notice.42@example.com"). Leave empty to let the SMTP server assign one.
	MessageID string
	// InReplyTo is the Message-ID of the mail being replied to
	InReplyTo string
	// References lists the Message-IDs of the thread, oldest first.
	// Defaults to []string{InReplyTo} when empty and InReplyTo is set.
	References []string
	// Headers adds extra headers such as "X-Notice-ID" or "Thread-Topic".
	// Headers managed by the library (From, To, Subject, Message-ID, ...) are rejected.
	Headers map[string]string
}

type Attachment struct {
	FileName string
	Data     []byte
}

type InlineImage struct {
	CID      string
	FileName string
	Data     []byte
}
