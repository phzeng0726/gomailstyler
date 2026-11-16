package mailstyler

type MailMessage struct {
	Subject string
	Message string
	To      []string
	Cc      []string
	// Bcc recipients receive the mail but never appear in it: they are only
	// added to the SMTP envelope, no Bcc header is written
	Bcc          []string
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
