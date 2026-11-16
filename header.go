package mailstyler

import (
	"fmt"
	"mime"
	"slices"
	"strings"
	"time"
)

// maxHeaderLineLength is the recommended line length from RFC 5322 section 2.1.1
const maxHeaderLineLength = 78

// reservedHeaders are written by the library and cannot be set via MailMessage.Headers
var reservedHeaders = map[string]struct{}{
	"from":                      {},
	"to":                        {},
	"cc":                        {},
	"bcc":                       {},
	"subject":                   {},
	"date":                      {},
	"message-id":                {},
	"in-reply-to":               {},
	"references":                {},
	"mime-version":              {},
	"content-type":              {},
	"content-transfer-encoding": {},
}

type headerField struct {
	name  string
	value string
}

func isReservedHeader(name string) bool {
	_, ok := reservedHeaders[strings.ToLower(name)]
	return ok
}

// validateHeaderValue rejects CR and LF, which would let a value start a new header
func validateHeaderValue(field, value string) error {
	if strings.ContainsAny(value, "\r\n") {
		return fmt.Errorf("%w: %s contains CR or LF", ErrInvalidHeader, field)
	}
	return nil
}

// validateHeaderName checks the RFC 5322 field-name rule: printable ASCII except ':'
func validateHeaderName(name string) error {
	if name == "" {
		return fmt.Errorf("%w: empty header name", ErrInvalidHeader)
	}
	for i := 0; i < len(name); i++ {
		if c := name[i]; c < 0x21 || c > 0x7e || c == ':' {
			return fmt.Errorf("%w: header name %q", ErrInvalidHeader, name)
		}
	}
	return nil
}

// validateFileName rejects values that would break the quoted filename parameter
func validateFileName(field, name string) error {
	if err := validateHeaderValue(field, name); err != nil {
		return err
	}
	if strings.Contains(name, `"`) {
		return fmt.Errorf("%w: %s contains a double quote", ErrInvalidHeader, field)
	}
	return nil
}

// validateContentID rejects values that would break the <cid> form of Content-ID
func validateContentID(field, cid string) error {
	if err := validateHeaderValue(field, cid); err != nil {
		return err
	}
	if strings.ContainsAny(cid, "<>") {
		return fmt.Errorf("%w: %s contains angle brackets", ErrInvalidHeader, field)
	}
	return nil
}

// normalizeMessageID validates id and returns it wrapped in angle brackets.
// Accepted input: "local@domain" or "<local@domain>" using printable ASCII
// without spaces or angle brackets, with exactly one '@'.
func normalizeMessageID(field, id string) (string, error) {
	inner := id
	if strings.HasPrefix(id, "<") && strings.HasSuffix(id, ">") {
		inner = id[1 : len(id)-1]
	}

	invalid := fmt.Errorf("%w: %s %q", ErrInvalidMessageID, field, id)

	local, domain, ok := strings.Cut(inner, "@")
	if !ok || local == "" || domain == "" || strings.Contains(domain, "@") {
		return "", invalid
	}
	for i := 0; i < len(inner); i++ {
		if c := inner[i]; c < 0x21 || c > 0x7e || c == '<' || c == '>' {
			return "", invalid
		}
	}

	return "<" + inner + ">", nil
}

// formatReferences joins normalized Message-IDs with spaces, folding with
// CRLF + space whenever the next ID would push the line past 78 characters
func formatReferences(ids []string) string {
	var b strings.Builder
	lineLen := len("References: ")

	for i, id := range ids {
		if i > 0 {
			if lineLen+1+len(id) > maxHeaderLineLength {
				b.WriteString("\r\n ")
				lineLen = 1
			} else {
				b.WriteByte(' ')
				lineLen++
			}
		}
		b.WriteString(id)
		lineLen += len(id)
	}

	return b.String()
}

// encodeHeaderValue applies RFC 2047 encoding to non-ASCII values; ASCII is returned unchanged
func encodeHeaderValue(value string) string {
	return mime.QEncoding.Encode("UTF-8", value)
}

// buildHeaders validates mm and returns the top-level headers in a fixed order
func (m *Manager) buildHeaders(mm MailMessage, boundary string) ([]headerField, error) {
	if err := validateHeaderValue("Subject", mm.Subject); err != nil {
		return nil, err
	}
	for i, addr := range mm.To {
		if err := validateHeaderValue(fmt.Sprintf("To[%d]", i), addr); err != nil {
			return nil, err
		}
	}
	for i, addr := range mm.Cc {
		if err := validateHeaderValue(fmt.Sprintf("Cc[%d]", i), addr); err != nil {
			return nil, err
		}
	}
	for i, att := range mm.Attachments {
		if err := validateFileName(fmt.Sprintf("Attachments[%d].FileName", i), att.FileName); err != nil {
			return nil, err
		}
	}
	for i, img := range mm.InlineImages {
		if err := validateFileName(fmt.Sprintf("InlineImages[%d].FileName", i), img.FileName); err != nil {
			return nil, err
		}
		if err := validateContentID(fmt.Sprintf("InlineImages[%d].CID", i), img.CID); err != nil {
			return nil, err
		}
	}

	threading, err := buildThreadingHeaders(mm)
	if err != nil {
		return nil, err
	}

	custom, err := buildCustomHeaders(mm.Headers)
	if err != nil {
		return nil, err
	}

	headers := []headerField{
		{"From", m.smtpSender},
		{"To", strings.Join(mm.To, ",")},
	}
	if len(mm.Cc) > 0 {
		headers = append(headers, headerField{"Cc", strings.Join(mm.Cc, ",")})
	}
	headers = append(headers,
		headerField{"Subject", encodeHeaderValue(mm.Subject)},
		headerField{"Date", m.now().Format(time.RFC1123Z)},
	)
	headers = append(headers, threading...)
	headers = append(headers, custom...)
	headers = append(headers,
		headerField{"MIME-Version", "1.0"},
		headerField{"Content-Type", fmt.Sprintf("multipart/mixed; boundary=%s", boundary)},
	)

	return headers, nil
}

// buildThreadingHeaders returns Message-ID, In-Reply-To and References, skipping empty ones
func buildThreadingHeaders(mm MailMessage) ([]headerField, error) {
	var headers []headerField

	if mm.MessageID != "" {
		id, err := normalizeMessageID("MessageID", mm.MessageID)
		if err != nil {
			return nil, err
		}
		headers = append(headers, headerField{"Message-ID", id})
	}

	var inReplyTo string
	if mm.InReplyTo != "" {
		id, err := normalizeMessageID("InReplyTo", mm.InReplyTo)
		if err != nil {
			return nil, err
		}
		inReplyTo = id
		headers = append(headers, headerField{"In-Reply-To", id})
	}

	refs := make([]string, 0, len(mm.References))
	for i, ref := range mm.References {
		id, err := normalizeMessageID(fmt.Sprintf("References[%d]", i), ref)
		if err != nil {
			return nil, err
		}
		refs = append(refs, id)
	}
	// A reply without explicit References points at its parent
	if len(refs) == 0 && inReplyTo != "" {
		refs = append(refs, inReplyTo)
	}
	if len(refs) > 0 {
		headers = append(headers, headerField{"References", formatReferences(refs)})
	}

	return headers, nil
}

// buildCustomHeaders validates user headers and returns them sorted case-insensitively
func buildCustomHeaders(custom map[string]string) ([]headerField, error) {
	names := make([]string, 0, len(custom))
	for name := range custom {
		names = append(names, name)
	}
	// Sort by lowercase name, then by original name so validation errors are deterministic
	slices.SortFunc(names, func(a, b string) int {
		if c := strings.Compare(strings.ToLower(a), strings.ToLower(b)); c != 0 {
			return c
		}
		return strings.Compare(a, b)
	})

	headers := make([]headerField, 0, len(names))
	seen := make(map[string]struct{}, len(names))

	for _, name := range names {
		if err := validateHeaderName(name); err != nil {
			return nil, err
		}
		if isReservedHeader(name) {
			return nil, fmt.Errorf("%w: %q", ErrReservedHeader, name)
		}

		lower := strings.ToLower(name)
		if _, dup := seen[lower]; dup {
			return nil, fmt.Errorf("%w: duplicate header name %q", ErrInvalidHeader, name)
		}
		seen[lower] = struct{}{}

		value := custom[name]
		if err := validateHeaderValue(fmt.Sprintf("Headers[%q]", name), value); err != nil {
			return nil, err
		}
		headers = append(headers, headerField{name, encodeHeaderValue(value)})
	}

	return headers, nil
}
