package mailstyler

import "errors"

var (
	// ErrInvalidHeader is returned when a header name or value is malformed,
	// e.g. a value contains CR or LF
	ErrInvalidHeader = errors.New("invalid header")

	// ErrReservedHeader is returned when MailMessage.Headers sets a header
	// that the library manages itself (From, To, Subject, Message-ID, ...)
	ErrReservedHeader = errors.New("reserved header")

	// ErrInvalidMessageID is returned when MessageID, InReplyTo or References
	// is not a valid Message-ID such as "<id@example.com>"
	ErrInvalidMessageID = errors.New("invalid message id")
)
