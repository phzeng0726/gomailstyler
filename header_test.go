package mailstyler

import (
	"mime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type HeaderSuite struct {
	suite.Suite
	m *Manager
}

func TestHeaderSuite(t *testing.T) {
	suite.Run(t, new(HeaderSuite))
}

func (s *HeaderSuite) SetupTest() {
	// buildHeaders only needs the sender and the clock
	s.m = &Manager{smtpSender: testSender, now: func() time.Time { return fixedNow }}
}

func (s *HeaderSuite) baseMail() MailMessage {
	return MailMessage{Subject: "Notice", Message: "<p>x</p>", To: []string{"a@example.com"}}
}

// headerMap flattens buildHeaders output for lookups; order is checked separately
func headerMap(headers []headerField) map[string]string {
	out := make(map[string]string, len(headers))
	for _, h := range headers {
		out[h.name] = h.value
	}
	return out
}

func headerNames(headers []headerField) []string {
	names := make([]string, 0, len(headers))
	for _, h := range headers {
		names = append(names, h.name)
	}
	return names
}

// --- normalizeMessageID ---

func (s *HeaderSuite) TestNormalizeMessageID_Valid() {
	cases := []struct{ in, want string }{
		{"a@example.com", "<a@example.com>"},
		{"<a@example.com>", "<a@example.com>"},
		{"maintenance.42.7@example.com", "<maintenance.42.7@example.com>"},
		{"x!#$%&'*+/=?^_`{|}~-@[127.0.0.1]", "<x!#$%&'*+/=?^_`{|}~-@[127.0.0.1]>"},
	}

	for _, tc := range cases {
		s.Run(tc.in, func() {
			got, err := normalizeMessageID("MessageID", tc.in)

			s.Require().NoError(err)
			s.Equal(tc.want, got)
		})
	}
}

func (s *HeaderSuite) TestNormalizeMessageID_Invalid() {
	cases := []struct{ name, in string }{
		{"empty", ""},
		{"missing at sign", "no-at-sign"},
		{"empty local part", "@example.com"},
		{"empty domain", "a@"},
		{"two at signs", "a@b@example.com"},
		{"space", "a b@example.com"},
		{"tab", "a\t@example.com"},
		{"non-ASCII", "維護@example.com"},
		{"only opening bracket", "<a@example.com"},
		{"only closing bracket", "a@example.com>"},
		{"double brackets", "<<a@example.com>>"},
		{"empty brackets", "<>"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			got, err := normalizeMessageID("MessageID", tc.in)

			s.ErrorIs(err, ErrInvalidMessageID)
			s.EqualError(err, `invalid message id: MessageID "`+strings.ReplaceAll(tc.in, "\t", `\t`)+`"`)
			s.Empty(got)
		})
	}
}

// --- validateHeaderName ---

func (s *HeaderSuite) TestValidateHeaderName() {
	s.NoError(validateHeaderName("X-Notice-ID"))
	s.NoError(validateHeaderName("!#$%&'*+-.^_`|~"))

	cases := []struct{ name, in, wantErr string }{
		{"empty", "", "invalid header: empty header name"},
		{"space", "Bad Name", `invalid header: header name "Bad Name"`},
		{"colon", "Bad:Name", `invalid header: header name "Bad:Name"`},
		{"control char", "X\tTab", `invalid header: header name "X\tTab"`},
		{"DEL", "X\x7f", `invalid header: header name "X\x7f"`},
		{"non-ASCII", "X-維護", `invalid header: header name "X-維護"`},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			err := validateHeaderName(tc.in)

			s.ErrorIs(err, ErrInvalidHeader)
			s.EqualError(err, tc.wantErr)
		})
	}
}

// --- formatReferences ---

func (s *HeaderSuite) TestFormatReferences_Single() {
	s.Equal("<a@example.com>", formatReferences([]string{"<a@example.com>"}))
}

func (s *HeaderSuite) TestFormatReferences_FoldsAtLineLimit() {
	// Each ID is 29 characters; "References: " is 12
	ids := []string{
		"<ref-01-abcdefgh@example.com>",
		"<ref-02-abcdefgh@example.com>",
		"<ref-03-abcdefgh@example.com>",
		"<ref-04-abcdefgh@example.com>",
	}
	s.Require().Len(ids[0], 29)

	got := formatReferences(ids)

	// 12+29+1+29 = 71 fits; adding the third would reach 101, so it folds
	s.Equal(ids[0]+" "+ids[1]+"\r\n "+ids[2]+" "+ids[3], got)
	for line := range strings.SplitSeq("References: "+got, "\r\n") {
		s.LessOrEqual(len(line), maxHeaderLineLength)
	}
}

func (s *HeaderSuite) TestFormatReferences_ExactlyAtLimitDoesNotFold() {
	first := "<ref-01-abcdefgh@example.com>"                    // 29
	fits := "<" + strings.Repeat("x", 22) + "@example.com>"     // 36: 12+29+1+36 = 78
	overflow := "<" + strings.Repeat("x", 23) + "@example.com>" // 37: 12+29+1+37 = 79
	s.Require().Len(fits, 36)
	s.Require().Len(overflow, 37)

	s.Equal(first+" "+fits, formatReferences([]string{first, fits}))
	s.Equal(first+"\r\n "+overflow, formatReferences([]string{first, overflow}))
}

// --- buildHeaders: order and defaults ---

func (s *HeaderSuite) TestBuildHeaders_MinimalOrder() {
	headers, err := s.m.buildHeaders(s.baseMail(), "BOUNDARY")

	s.Require().NoError(err)
	s.Equal([]headerField{
		{"From", testSender},
		{"To", "a@example.com"},
		{"Subject", "Notice"},
		{"Date", testDate},
		{"MIME-Version", "1.0"},
		{"Content-Type", "multipart/mixed; boundary=BOUNDARY"},
	}, headers)
}

func (s *HeaderSuite) TestBuildHeaders_FullOrder() {
	mm := s.baseMail()
	mm.Cc = []string{"c@example.com"}
	mm.MessageID = "id@example.com"
	mm.InReplyTo = "parent@example.com"
	mm.Headers = map[string]string{"X-B": "2", "X-A": "1"}

	headers, err := s.m.buildHeaders(mm, "BOUNDARY")

	s.Require().NoError(err)
	s.Equal([]string{
		"From", "To", "Cc", "Subject", "Date",
		"Message-ID", "In-Reply-To", "References",
		"X-A", "X-B",
		"MIME-Version", "Content-Type",
	}, headerNames(headers))
}

// --- buildHeaders: Bcc and empty To ---

func (s *HeaderSuite) TestBuildHeaders_BccNeverWritten() {
	mm := s.baseMail()
	mm.Bcc = []string{"secret@example.com"}

	headers, err := s.m.buildHeaders(mm, "BOUNDARY")

	s.Require().NoError(err)
	s.Equal([]string{"From", "To", "Subject", "Date", "MIME-Version", "Content-Type"}, headerNames(headers))
	for _, h := range headers {
		s.NotContains(h.value, "secret@example.com", "Bcc address leaked into %s", h.name)
	}
}

func (s *HeaderSuite) TestBuildHeaders_EmptyToOmitted() {
	mm := MailMessage{Subject: "Notice", Message: "<p>x</p>", Bcc: []string{"x@example.com"}}

	headers, err := s.m.buildHeaders(mm, "BOUNDARY")

	s.Require().NoError(err)
	s.Equal([]headerField{
		{"From", testSender},
		{"Subject", "Notice"},
		{"Date", testDate},
		{"MIME-Version", "1.0"},
		{"Content-Type", "multipart/mixed; boundary=BOUNDARY"},
	}, headers)
}

// --- buildHeaders: threading ---

func (s *HeaderSuite) TestBuildHeaders_InReplyToDefaultsReferences() {
	mm := s.baseMail()
	mm.InReplyTo = "parent@example.com"

	headers, err := s.m.buildHeaders(mm, "B")

	s.Require().NoError(err)
	got := headerMap(headers)
	s.Equal("<parent@example.com>", got["In-Reply-To"])
	s.Equal("<parent@example.com>", got["References"])
	_, hasMessageID := got["Message-ID"]
	s.False(hasMessageID)
}

func (s *HeaderSuite) TestBuildHeaders_ExplicitReferencesKeepOrder() {
	mm := s.baseMail()
	mm.InReplyTo = "b@example.com"
	mm.References = []string{"a@example.com", "<b@example.com>"}

	headers, err := s.m.buildHeaders(mm, "B")

	s.Require().NoError(err)
	got := headerMap(headers)
	s.Equal("<b@example.com>", got["In-Reply-To"])
	s.Equal("<a@example.com> <b@example.com>", got["References"])
}

func (s *HeaderSuite) TestBuildHeaders_ReferencesWithoutInReplyTo() {
	mm := s.baseMail()
	mm.References = []string{"a@example.com"}

	headers, err := s.m.buildHeaders(mm, "B")

	s.Require().NoError(err)
	got := headerMap(headers)
	s.Equal("<a@example.com>", got["References"])
	_, hasInReplyTo := got["In-Reply-To"]
	s.False(hasInReplyTo)
}

func (s *HeaderSuite) TestBuildHeaders_InvalidThreadingIDs() {
	cases := []struct {
		name    string
		mutate  func(*MailMessage)
		wantErr string
	}{
		{"MessageID", func(mm *MailMessage) { mm.MessageID = "no-at-sign" }, `invalid message id: MessageID "no-at-sign"`},
		{"InReplyTo", func(mm *MailMessage) { mm.InReplyTo = "a b@example.com" }, `invalid message id: InReplyTo "a b@example.com"`},
		{"References index", func(mm *MailMessage) { mm.References = []string{"ok@example.com", "a@b@example.com"} }, `invalid message id: References[1] "a@b@example.com"`},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			mm := s.baseMail()
			tc.mutate(&mm)

			headers, err := s.m.buildHeaders(mm, "B")

			s.ErrorIs(err, ErrInvalidMessageID)
			s.EqualError(err, tc.wantErr)
			s.Nil(headers)
		})
	}
}

// --- buildHeaders: custom headers ---

func (s *HeaderSuite) TestBuildHeaders_CustomHeadersKeepCaseAndSort() {
	mm := s.baseMail()
	mm.Headers = map[string]string{"x-lower": "1", "X-Upper": "2", "Thread-Topic": "Maintenance"}

	headers, err := s.m.buildHeaders(mm, "B")

	s.Require().NoError(err)
	s.Equal([]headerField{
		{"Thread-Topic", "Maintenance"},
		{"x-lower", "1"},
		{"X-Upper", "2"},
	}, headers[4:7])
}

func (s *HeaderSuite) TestBuildHeaders_NonASCIICustomValueEncoded() {
	mm := s.baseMail()
	mm.Headers = map[string]string{"Thread-Topic": "維護通知"}

	headers, err := s.m.buildHeaders(mm, "B")

	s.Require().NoError(err)
	encoded := headerMap(headers)["Thread-Topic"]
	s.True(strings.HasPrefix(encoded, "=?UTF-8?q?"), encoded)
	decoded, err := new(mime.WordDecoder).DecodeHeader(encoded)
	s.Require().NoError(err)
	s.Equal("維護通知", decoded)
}

func (s *HeaderSuite) TestBuildHeaders_ReservedHeaderRejected() {
	names := []string{
		"From", "to", "CC", "Bcc", "subject", "Date", "Message-Id", "in-reply-to",
		"References", "mime-version", "Content-Type", "content-transfer-encoding",
	}

	for _, name := range names {
		s.Run(name, func() {
			mm := s.baseMail()
			mm.Headers = map[string]string{name: "x"}

			headers, err := s.m.buildHeaders(mm, "B")

			s.ErrorIs(err, ErrReservedHeader)
			s.EqualError(err, `reserved header: "`+name+`"`)
			s.Nil(headers)
		})
	}
}

func (s *HeaderSuite) TestBuildHeaders_CustomHeaderErrors() {
	cases := []struct {
		name    string
		headers map[string]string
		wantErr string
	}{
		{"invalid name", map[string]string{"Bad Name": "x"}, `invalid header: header name "Bad Name"`},
		{"empty name", map[string]string{"": "x"}, "invalid header: empty header name"},
		{"duplicate ignoring case", map[string]string{"X-Tag": "1", "x-tag": "2"}, `invalid header: duplicate header name "x-tag"`},
		{"CRLF value", map[string]string{"X-Tag": "a\r\nBcc: evil@example.com"}, `invalid header: Headers["X-Tag"] contains CR or LF`},
		{"LF only value", map[string]string{"X-Tag": "a\nb"}, `invalid header: Headers["X-Tag"] contains CR or LF`},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			mm := s.baseMail()
			mm.Headers = tc.headers

			headers, err := s.m.buildHeaders(mm, "B")

			s.ErrorIs(err, ErrInvalidHeader)
			s.EqualError(err, tc.wantErr)
			s.Nil(headers)
		})
	}
}

// --- buildHeaders: CR/LF and parameter validation of other fields ---

func (s *HeaderSuite) TestBuildHeaders_FieldValidation() {
	cases := []struct {
		name    string
		mutate  func(*MailMessage)
		wantErr string
	}{
		{"Subject CR only", func(mm *MailMessage) { mm.Subject = "a\rb" }, "invalid header: Subject contains CR or LF"},
		{"To", func(mm *MailMessage) { mm.To = []string{"a@example.com", "b@example.com\nBcc: x"} }, "invalid header: To[1] contains CR or LF"},
		{"Cc", func(mm *MailMessage) { mm.Cc = []string{"c@example.com\r\n"} }, "invalid header: Cc[0] contains CR or LF"},
		{"Bcc", func(mm *MailMessage) { mm.Bcc = []string{"x@example.com", "y@example.com\r\nX-Evil: 1"} }, "invalid header: Bcc[1] contains CR or LF"},
		{"attachment name CRLF", func(mm *MailMessage) {
			mm.Attachments = []Attachment{{FileName: "a\r\n.txt"}}
		}, "invalid header: Attachments[0].FileName contains CR or LF"},
		{"attachment name quote", func(mm *MailMessage) {
			mm.Attachments = []Attachment{{FileName: "ok.txt"}, {FileName: `a".txt`}}
		}, "invalid header: Attachments[1].FileName contains a double quote"},
		{"inline name quote", func(mm *MailMessage) {
			mm.InlineImages = []InlineImage{{CID: "logo", FileName: `l"ogo.png`}}
		}, "invalid header: InlineImages[0].FileName contains a double quote"},
		{"inline CID CRLF", func(mm *MailMessage) {
			mm.InlineImages = []InlineImage{{CID: "lo\ngo", FileName: "logo.png"}}
		}, "invalid header: InlineImages[0].CID contains CR or LF"},
		{"inline CID brackets", func(mm *MailMessage) {
			mm.InlineImages = []InlineImage{{CID: "<logo>", FileName: "logo.png"}}
		}, "invalid header: InlineImages[0].CID contains angle brackets"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			mm := s.baseMail()
			tc.mutate(&mm)

			headers, err := s.m.buildHeaders(mm, "B")

			s.ErrorIs(err, ErrInvalidHeader)
			s.EqualError(err, tc.wantErr)
			s.Nil(headers)
		})
	}
}

func (s *HeaderSuite) TestBuildHeaders_ValidAttachmentsAccepted() {
	mm := s.baseMail()
	mm.Attachments = []Attachment{{FileName: "報告 2026.pdf"}}
	mm.InlineImages = []InlineImage{{CID: "logo@example", FileName: "logo.png"}}

	_, err := s.m.buildHeaders(mm, "B")

	s.NoError(err)
}

// --- isReservedHeader / encodeHeaderValue ---

func (s *HeaderSuite) TestIsReservedHeader() {
	s.True(isReservedHeader("MESSAGE-ID"))
	s.False(isReservedHeader("X-Message-ID"))
	s.False(isReservedHeader("Thread-Topic"))
}

func (s *HeaderSuite) TestEncodeHeaderValue_ASCIIUnchanged() {
	s.Equal("Maintenance notice", encodeHeaderValue("Maintenance notice"))
	s.Equal("", encodeHeaderValue(""))
}
