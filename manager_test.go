package mailstyler

import (
	"bytes"
	"encoding/base64"
	"errors"
	"html/template"
	"io"
	"log"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/textproto"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/suite"
)

const (
	testdataPath = "internal/service/testdata"
	testSender   = "sender@example.com"
	// testDate is fixedNow formatted as RFC 1123 with numeric zone
	testDate = "Mon, 16 Nov 2026 09:30:00 +0800"
)

// fixedNow is injected as Manager.now so the Date header is predictable
var fixedNow = time.Date(2026, 11, 16, 9, 30, 0, 0, time.FixedZone("CST", 8*60*60))

// pngHeader is enough for http.DetectContentType to report image/png
var pngHeader = []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR")

var (
	boundaryPattern      = regexp.MustCompile(`^[0-9a-f]{60}$`)
	boundaryParamPattern = regexp.MustCompile(`boundary=[0-9a-f]{60}`)
)

type rawPart struct {
	header textproto.MIMEHeader
	body   string
}

type parsedMessage struct {
	header   mail.Header
	boundary string
	parts    []rawPart
}

type ManagerSuite struct {
	suite.Suite
}

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(ManagerSuite))
}

func (s *ManagerSuite) SetupSuite() {
	log.SetOutput(io.Discard)
}

func (s *ManagerSuite) TearDownSuite() {
	log.SetOutput(os.Stderr)
}

func (s *ManagerSuite) newManager(opts ...ManagerOption) *Manager {
	s.T().Helper()

	m, err := NewManager(opts...)
	s.Require().NoError(err)
	m.now = func() time.Time { return fixedNow }
	return m
}

func (s *ManagerSuite) newManagerForServer(srv *fakeSMTPServer, opts ...ManagerOption) *Manager {
	s.T().Helper()

	return s.newManager(append([]ManagerOption{WithSMTP("127.0.0.1", srv.Port(), testSender)}, opts...)...)
}

// build calls buildHTMLMessage and requires it to succeed
func (s *ManagerSuite) build(m *Manager, mm MailMessage) []byte {
	s.T().Helper()

	raw, err := m.buildHTMLMessage(mm)
	s.Require().NoError(err)
	return raw
}

// parseMessage splits a MIME message into headers and raw (undecoded) parts
func (s *ManagerSuite) parseMessage(raw []byte) parsedMessage {
	s.T().Helper()

	msg, err := mail.ReadMessage(bytes.NewReader(raw))
	s.Require().NoError(err)

	mediaType, params, err := mime.ParseMediaType(msg.Header.Get("Content-Type"))
	s.Require().NoError(err)
	s.Require().Equal("multipart/mixed", mediaType)

	reader := multipart.NewReader(msg.Body, params["boundary"])
	var parts []rawPart
	for {
		p, err := reader.NextRawPart()
		if errors.Is(err, io.EOF) {
			break
		}
		s.Require().NoError(err)

		body, err := io.ReadAll(p)
		s.Require().NoError(err)
		parts = append(parts, rawPart{header: p.Header, body: string(body)})
	}

	return parsedMessage{header: msg.Header, boundary: params["boundary"], parts: parts}
}

// headerBlock returns the raw top-level header section with the boundary replaced by "BOUNDARY"
func headerBlock(raw []byte) string {
	block, _, _ := bytes.Cut(raw, []byte("\r\n\r\n"))
	return boundaryParamPattern.ReplaceAllString(string(block)+"\r\n", "boundary=BOUNDARY")
}

// base64Lines returns the non-empty lines of a base64 part body
func base64Lines(body string) []string {
	return strings.FieldsFunc(body, func(r rune) bool { return r == '\r' || r == '\n' })
}

func decodeBase64Part(body string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.Join(base64Lines(body), ""))
}

func (s *ManagerSuite) decodeQP(body string) string {
	s.T().Helper()

	decoded, err := io.ReadAll(quotedprintable.NewReader(strings.NewReader(body)))
	s.Require().NoError(err)
	return string(decoded)
}

// --- NewManager ---

func (s *ManagerSuite) TestNewManager_MissingRequiredField() {
	cases := []struct {
		name               string
		host, port, sender string
		wantErr            string
	}{
		{"empty host", "", "587", testSender, "invalid manager config: empty smtpHost"},
		{"empty port", "smtp.example.com", "", testSender, "invalid manager config: empty smtpPort"},
		{"empty sender", "smtp.example.com", "587", "", "invalid manager config: empty smtpSender"},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			m, err := NewManager(WithSMTP(tc.host, tc.port, tc.sender))

			s.EqualError(err, tc.wantErr)
			s.Nil(m)
		})
	}
}

func (s *ManagerSuite) TestNewManager_SenderWithCRLF() {
	m, err := NewManager(WithSMTP("smtp.example.com", "587", "sender@example.com\r\nBcc: evil@example.com"))

	s.ErrorIs(err, ErrInvalidHeader)
	s.EqualError(err, "invalid manager config: invalid header: smtpSender contains CR or LF")
	s.Nil(m)
}

func (s *ManagerSuite) TestNewManager_NoOptions() {
	m, err := NewManager()

	s.EqualError(err, "invalid manager config: empty smtpHost")
	s.Nil(m)
}

func (s *ManagerSuite) TestNewManager_MinimalOptions() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender))

	s.Equal("smtp.example.com", m.smtpHost)
	s.Equal("587", m.smtpPort)
	s.Equal(testSender, m.smtpSender)
	s.Nil(m.smtpPassword)
	s.NotNil(m.tmplSvc)
	s.NotNil(m.cssToolSvc)
}

func (s *ManagerSuite) TestNewManager_DefaultClockIsTimeNow() {
	m, err := NewManager(WithSMTP("smtp.example.com", "587", testSender))
	s.Require().NoError(err)

	s.Require().NotNil(m.now)
	s.WithinDuration(time.Now(), m.now(), time.Second)
}

func (s *ManagerSuite) TestNewManager_AllOptions() {
	m := s.newManager(
		WithSMTP("smtp.example.com", "587", testSender),
		WithSMTPPassword("secret"),
		WithTemplatePath(testdataPath),
		WithCSSPath(testdataPath),
	)

	s.Equal("smtp.example.com", m.smtpHost)
	s.Equal("587", m.smtpPort)
	s.Equal(testSender, m.smtpSender)
	s.Require().NotNil(m.smtpPassword)
	s.Equal("secret", *m.smtpPassword)

	// Template and CSS paths are verified through rendering
	got, err := m.RenderTemplateWithCSS("no_head.html", "styles.css", map[string]string{"Name": "Rita"})
	s.Require().NoError(err)
	s.Equal(`<html><head></head><body><p style="color:red;font-weight:bold">Rita</p></body></html>`, got)
}

func (s *ManagerSuite) TestNewManager_EmptyPasswordStillSet() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender), WithSMTPPassword(""))

	s.Require().NotNil(m.smtpPassword)
	s.Equal("", *m.smtpPassword)
}

// --- buildHTMLMessage ---

func (s *ManagerSuite) TestBuildHTMLMessage_Minimal() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender))

	raw := s.build(m, MailMessage{
		Subject: "Maintenance notice",
		Message: "<p>Hello</p>",
		To:      []string{"a@example.com"},
	})

	s.Equal("From: sender@example.com\r\n"+
		"To: a@example.com\r\n"+
		"Subject: Maintenance notice\r\n"+
		"Date: "+testDate+"\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: multipart/mixed; boundary=BOUNDARY\r\n", headerBlock(raw))

	msg := s.parseMessage(raw)
	s.Regexp(boundaryPattern, msg.boundary)

	s.Require().Len(msg.parts, 1)
	s.Equal(textproto.MIMEHeader{
		"Content-Type":              {`text/html; charset="UTF-8"`},
		"Content-Transfer-Encoding": {"quoted-printable"},
	}, msg.parts[0].header)
	s.Equal("<p>Hello</p>", s.decodeQP(msg.parts[0].body))
}

func (s *ManagerSuite) TestBuildHTMLMessage_MultipleRecipientsAndCc() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender))

	msg := s.parseMessage(s.build(m, MailMessage{
		Subject: "Notice",
		Message: "<p>Hi</p>",
		To:      []string{"a@example.com", "b@example.com"},
		Cc:      []string{"c@example.com", "d@example.com"},
	}))

	s.Len(msg.header, 7)
	s.Equal("a@example.com,b@example.com", msg.header.Get("To"))
	s.Equal("c@example.com,d@example.com", msg.header.Get("Cc"))
}

func (s *ManagerSuite) TestBuildHTMLMessage_FullThreadingHeaderBlock() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender))

	raw := s.build(m, MailMessage{
		Subject:   "RE: Maintenance notice",
		Message:   "<p>Done</p>",
		To:        []string{"a@example.com"},
		Cc:        []string{"c@example.com"},
		MessageID: "done.42.7@example.com",
		InReplyTo: "<maintenance.42.7@example.com>",
		Headers: map[string]string{
			"X-Notice-ID":  "42",
			"Thread-Topic": "Maintenance notice",
		},
	})

	s.Equal("From: sender@example.com\r\n"+
		"To: a@example.com\r\n"+
		"Cc: c@example.com\r\n"+
		"Subject: RE: Maintenance notice\r\n"+
		"Date: "+testDate+"\r\n"+
		"Message-ID: <done.42.7@example.com>\r\n"+
		"In-Reply-To: <maintenance.42.7@example.com>\r\n"+
		"References: <maintenance.42.7@example.com>\r\n"+
		"Thread-Topic: Maintenance notice\r\n"+
		"X-Notice-ID: 42\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: multipart/mixed; boundary=BOUNDARY\r\n", headerBlock(raw))
}

func (s *ManagerSuite) TestBuildHTMLMessage_FoldedReferencesParse() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender))
	refs := []string{
		"ref-01-abcdefgh@example.com",
		"ref-02-abcdefgh@example.com",
		"<ref-03-abcdefgh@example.com>",
		"ref-04-abcdefgh@example.com",
	}

	raw := s.build(m, MailMessage{
		Subject:    "RE: Folded",
		Message:    "<p>x</p>",
		To:         []string{"a@example.com"},
		References: refs,
	})

	// The raw header is folded onto a continuation line...
	s.Contains(headerBlock(raw), "References: <ref-01-abcdefgh@example.com> <ref-02-abcdefgh@example.com>\r\n <ref-03-abcdefgh@example.com> <ref-04-abcdefgh@example.com>\r\n")
	// ...and unfolds to the space-separated ID list when parsed
	s.Equal("<ref-01-abcdefgh@example.com> <ref-02-abcdefgh@example.com> <ref-03-abcdefgh@example.com> <ref-04-abcdefgh@example.com>",
		s.parseMessage(raw).header.Get("References"))
}

func (s *ManagerSuite) TestBuildHTMLMessage_HeaderOrderIsStable() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender))
	mm := MailMessage{
		Subject: "Order",
		Message: "<p>x</p>",
		To:      []string{"a@example.com"},
		Headers: map[string]string{"X-C": "3", "x-a": "1", "X-B": "2", "Thread-Topic": "t"},
	}

	first := headerBlock(s.build(m, mm))
	for range 20 {
		s.Equal(first, headerBlock(s.build(m, mm)))
	}
	s.Contains(first, "Thread-Topic: t\r\nx-a: 1\r\nX-B: 2\r\nX-C: 3\r\nMIME-Version: 1.0\r\n")
}

func (s *ManagerSuite) TestBuildHTMLMessage_AttachmentAndInlineImage() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender))
	attachment := []byte("hello attachment")

	msg := s.parseMessage(s.build(m, MailMessage{
		Subject:      "With files",
		Message:      `<img src="cid:logo">`,
		To:           []string{"a@example.com"},
		Attachments:  []Attachment{{FileName: "report.txt", Data: attachment}},
		InlineImages: []InlineImage{{CID: "logo", FileName: "logo.png", Data: pngHeader}},
	}))

	s.Require().Len(msg.parts, 3)

	// Part order: HTML, attachments, inline images
	s.Equal(`<img src="cid:logo">`, s.decodeQP(msg.parts[0].body))

	att := msg.parts[1]
	s.Equal(textproto.MIMEHeader{
		"Content-Type":              {"text/plain; charset=utf-8"},
		"Content-Transfer-Encoding": {"base64"},
		"Content-Disposition":       {`attachment; filename="report.txt"`},
	}, att.header)
	decoded, err := decodeBase64Part(att.body)
	s.Require().NoError(err)
	s.Equal(attachment, decoded)

	img := msg.parts[2]
	s.Equal(textproto.MIMEHeader{
		"Content-Type":              {"image/png"},
		"Content-Transfer-Encoding": {"base64"},
		"Content-Disposition":       {`inline; filename="logo.png"`},
		"Content-Id":                {"<logo>"},
	}, img.header)
	decoded, err = decodeBase64Part(img.body)
	s.Require().NoError(err)
	s.Equal(pngHeader, decoded)
}

func (s *ManagerSuite) TestBuildHTMLMessage_MultipleAttachmentsKeepOrder() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender))

	msg := s.parseMessage(s.build(m, MailMessage{
		Subject:     "Files",
		Message:     "<p>x</p>",
		To:          []string{"a@example.com"},
		Attachments: []Attachment{{FileName: "1.txt", Data: []byte("one")}, {FileName: "2.txt", Data: []byte("two")}},
	}))

	s.Require().Len(msg.parts, 3)
	s.Equal(`attachment; filename="1.txt"`, msg.parts[1].header.Get("Content-Disposition"))
	s.Equal(`attachment; filename="2.txt"`, msg.parts[2].header.Get("Content-Disposition"))
}

func (s *ManagerSuite) TestBuildHTMLMessage_Base64LineLength() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender))

	// 57 bytes encode to exactly 76 base64 characters
	cases := []struct {
		name      string
		size      int
		wantLines []int
	}{
		{"empty", 0, nil},
		{"shorter than one line", 10, []int{16}},
		{"exactly one line", 57, []int{76}},
		{"one byte over one line", 58, []int{76, 4}},
		{"exactly two lines", 114, []int{76, 76}},
		{"partial third line", 120, []int{76, 76, 8}},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			data := bytes.Repeat([]byte("a"), tc.size)

			msg := s.parseMessage(s.build(m, MailMessage{
				Subject:     "Base64",
				Message:     "<p>x</p>",
				To:          []string{"a@example.com"},
				Attachments: []Attachment{{FileName: "a.txt", Data: data}},
			}))
			s.Require().Len(msg.parts, 2)

			var gotLines []int
			for _, line := range base64Lines(msg.parts[1].body) {
				gotLines = append(gotLines, len(line))
			}
			s.Equal(tc.wantLines, gotLines)

			// Every line is terminated by CRLF
			s.Equal(len(tc.wantLines), strings.Count(msg.parts[1].body, "\r\n"))
		})
	}
}

func (s *ManagerSuite) TestBuildHTMLMessage_LongNonASCIIBodyIsQuotedPrintable() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender))
	body := strings.Repeat("<p>維護完成通知</p>", 100) // single line, well over 998 bytes
	s.Require().Greater(len(body), 998)

	msg := s.parseMessage(s.build(m, MailMessage{
		Subject: "QP",
		Message: body,
		To:      []string{"a@example.com"},
	}))
	s.Require().Len(msg.parts, 1)

	raw := msg.parts[0].body
	for line := range strings.SplitSeq(raw, "\r\n") {
		s.LessOrEqual(len(line), 76, "line too long: %q", line)
	}
	for i := 0; i < len(raw); i++ {
		s.Require().Less(raw[i], byte(0x80), "non-ASCII byte at %d", i)
	}
	s.Equal(body, s.decodeQP(raw))
}

func (s *ManagerSuite) TestBuildHTMLMessage_SubjectWithCRLFRejected() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender))

	raw, err := m.buildHTMLMessage(MailMessage{
		Subject: "Hello\r\nBcc: evil@example.com",
		Message: "<p>x</p>",
		To:      []string{"a@example.com"},
	})

	s.ErrorIs(err, ErrInvalidHeader)
	s.EqualError(err, "invalid header: Subject contains CR or LF")
	s.Nil(raw)
}

func (s *ManagerSuite) TestBuildHTMLMessage_NonASCIISubjectEncoded() {
	m := s.newManager(WithSMTP("smtp.example.com", "587", testSender))
	subject := "RE: 維護完成通知"

	raw := s.build(m, MailMessage{
		Subject: subject,
		Message: "<p>x</p>",
		To:      []string{"a@example.com"},
	})

	header := headerBlock(raw)
	s.Contains(header, "Subject: =?UTF-8?q?")
	for i := 0; i < len(header); i++ {
		s.Require().Less(header[i], byte(0x80), "non-ASCII byte in header at %d", i)
	}

	encoded := s.parseMessage(raw).header.Get("Subject")
	decoded, err := new(mime.WordDecoder).DecodeHeader(encoded)
	s.Require().NoError(err)
	s.Equal(subject, decoded)
}

// --- SendMail ---

func (s *ManagerSuite) sampleMail() MailMessage {
	return MailMessage{
		Subject: "Maintenance notice",
		Message: "<p>Scheduled maintenance</p>",
		To:      []string{"a@example.com", "b@example.com"},
		Cc:      []string{"c@example.com"},
	}
}

func (s *ManagerSuite) TestSendMail_WithoutPassword() {
	srv := newFakeSMTPServer(s.T(), fakeSMTPOptions{advertiseAuth: true})
	m := s.newManagerForServer(srv)

	s.Require().NoError(m.SendMail(s.sampleMail()))

	rec := srv.Record()
	s.Equal(1, rec.connections)
	s.Equal(testSender, rec.mailFrom)
	s.Equal([]string{"a@example.com", "b@example.com", "c@example.com"}, rec.rcpts)
	s.False(rec.authCalled, "AUTH must not be sent without a password")

	msg := s.parseMessage([]byte(rec.data))
	s.Equal(testSender, msg.header.Get("From"))
	s.Equal("a@example.com,b@example.com", msg.header.Get("To"))
	s.Equal("c@example.com", msg.header.Get("Cc"))
	s.Equal("Maintenance notice", msg.header.Get("Subject"))
	s.Equal(testDate, msg.header.Get("Date"))
	s.Require().Len(msg.parts, 1)
	s.Equal("<p>Scheduled maintenance</p>", s.decodeQP(msg.parts[0].body))
}

func (s *ManagerSuite) TestSendMail_ThreadingHeaders() {
	srv := newFakeSMTPServer(s.T(), fakeSMTPOptions{advertiseAuth: true})
	m := s.newManagerForServer(srv)

	mm := s.sampleMail()
	mm.Subject = "RE: Maintenance notice"
	mm.MessageID = "done.42.7@example.com"
	mm.InReplyTo = "maintenance.42.7@example.com"
	mm.References = []string{"maintenance.42.7@example.com"}

	s.Require().NoError(m.SendMail(mm))

	msg := s.parseMessage([]byte(srv.Record().data))
	s.Equal("RE: Maintenance notice", msg.header.Get("Subject"))
	s.Equal("<done.42.7@example.com>", msg.header.Get("Message-ID"))
	s.Equal("<maintenance.42.7@example.com>", msg.header.Get("In-Reply-To"))
	s.Equal("<maintenance.42.7@example.com>", msg.header.Get("References"))
}

func (s *ManagerSuite) TestSendMail_EmptyPasswordSkipsAuth() {
	srv := newFakeSMTPServer(s.T(), fakeSMTPOptions{advertiseAuth: true})
	m := s.newManagerForServer(srv, WithSMTPPassword(""))

	s.Require().NoError(m.SendMail(s.sampleMail()))

	rec := srv.Record()
	s.False(rec.authCalled, "AUTH must not be sent with an empty password")
	s.Equal([]string{"a@example.com", "b@example.com", "c@example.com"}, rec.rcpts)
}

func (s *ManagerSuite) TestSendMail_WithPassword() {
	srv := newFakeSMTPServer(s.T(), fakeSMTPOptions{advertiseAuth: true})
	m := s.newManagerForServer(srv, WithSMTPPassword("secret"))

	s.Require().NoError(m.SendMail(s.sampleMail()))

	rec := srv.Record()
	s.True(rec.authCalled)
	s.Equal(testSender, rec.authUser)
	s.Equal("secret", rec.authPass)
	s.Equal(testSender, rec.mailFrom)
	s.Equal([]string{"a@example.com", "b@example.com", "c@example.com"}, rec.rcpts)
}

func (s *ManagerSuite) TestSendMail_PasswordButServerWithoutAuth() {
	srv := newFakeSMTPServer(s.T(), fakeSMTPOptions{advertiseAuth: false})
	m := s.newManagerForServer(srv, WithSMTPPassword("secret"))

	err := m.SendMail(s.sampleMail())

	s.EqualError(err, "smtp: server doesn't support AUTH")
	s.Equal("", srv.Record().data)
}

func (s *ManagerSuite) TestSendMail_RcptRejected() {
	srv := newFakeSMTPServer(s.T(), fakeSMTPOptions{advertiseAuth: true, rejectRcpt: true})
	m := s.newManagerForServer(srv)

	err := m.SendMail(s.sampleMail())

	protoErr, ok := errors.AsType[*textproto.Error](err)
	s.Require().True(ok, "expected *textproto.Error, got %T: %v", err, err)
	s.Equal(550, protoErr.Code)
	s.Equal("5.1.1 mailbox unavailable", protoErr.Msg)
	s.Equal("", srv.Record().data)
}

func (s *ManagerSuite) TestSendMail_ConnectionRefused() {
	// Reserve a free port, then close it so nothing is listening
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	s.Require().NoError(err)
	addr := listener.Addr().(*net.TCPAddr)
	s.Require().NoError(listener.Close())

	m := s.newManager(WithSMTP("127.0.0.1", strconv.Itoa(addr.Port), testSender))

	err = m.SendMail(s.sampleMail())

	opErr, ok := errors.AsType[*net.OpError](err)
	s.Require().True(ok, "expected *net.OpError, got %T: %v", err, err)
	s.Equal("dial", opErr.Op)
}

func (s *ManagerSuite) TestSendMail_ValidationFailsBeforeConnecting() {
	srv := newFakeSMTPServer(s.T(), fakeSMTPOptions{advertiseAuth: true})
	m := s.newManagerForServer(srv)

	mm := s.sampleMail()
	mm.Headers = map[string]string{"Subject": "override"}

	err := m.SendMail(mm)

	s.ErrorIs(err, ErrReservedHeader)
	s.EqualError(err, `reserved header: "Subject"`)
	s.Equal(0, srv.Record().connections)
}

func (s *ManagerSuite) TestSendMail_DoesNotModifyCallerToSlice() {
	srv := newFakeSMTPServer(s.T(), fakeSMTPOptions{advertiseAuth: true})
	m := s.newManagerForServer(srv)

	backing := make([]string, 2)
	backing[0] = "a@example.com"
	backing[1] = "untouched"
	to := backing[:1] // len 1, cap 2

	s.Require().NoError(m.SendMail(MailMessage{
		Subject: "Alias",
		Message: "<p>x</p>",
		To:      to,
		Cc:      []string{"c@example.com"},
	}))

	s.Equal("untouched", backing[1])
	s.Equal([]string{"a@example.com", "c@example.com"}, srv.Record().rcpts)
}

// --- Render delegation ---

func (s *ManagerSuite) renderManager() *Manager {
	return s.newManager(
		WithSMTP("smtp.example.com", "587", testSender),
		WithTemplatePath(testdataPath),
		WithCSSPath(testdataPath),
	)
}

func (s *ManagerSuite) customFuncs() template.FuncMap {
	return template.FuncMap{
		"toUpper": func(v string) string { return "custom:" + v },
		"greet":   func(v string) string { return "hi " + v },
	}
}

func (s *ManagerSuite) assertStyledParagraph(html, wantText string) {
	s.T().Helper()

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	s.Require().NoError(err)

	p := doc.Find("body p").First()
	s.Equal(wantText, p.Text())
	style, _ := p.Attr("style")
	s.Equal("color:red;font-weight:bold", style)
	_, hasClass := p.Attr("class")
	s.False(hasClass)
}

func (s *ManagerSuite) TestRenderTemplate() {
	got, err := s.renderManager().RenderTemplate("basic.html", map[string]string{"Name": "Rita"})

	s.Require().NoError(err)
	s.Equal("<p>Hello, Rita!</p>", got)
}

func (s *ManagerSuite) TestRenderTemplateWithFuncs_PassesCustomFuncs() {
	got, err := s.renderManager().RenderTemplateWithFuncs("custom_funcs.html", map[string]string{"Name": "Rita"}, s.customFuncs())

	s.Require().NoError(err)
	s.Equal("custom:Rita|hi Rita", got)
}

func (s *ManagerSuite) TestRenderTemplateWithFuncs_WithoutCustomFuncs() {
	got, err := s.renderManager().RenderTemplateWithFuncs("custom_funcs.html", map[string]string{"Name": "Rita"})

	s.EqualError(err, `template: custom_funcs.html:1: function "greet" not defined`)
	s.Empty(got)
}

func (s *ManagerSuite) TestRenderTemplateWithCSS() {
	got, err := s.renderManager().RenderTemplateWithCSS("styled.html", "styles.css", map[string]string{"Name": "Rita"})

	s.Require().NoError(err)
	s.assertStyledParagraph(got, "Rita")
}

func (s *ManagerSuite) TestRenderTemplateWithFuncsAndCSS_PassesCustomFuncs() {
	got, err := s.renderManager().RenderTemplateWithFuncsAndCSS("styled_funcs.html", "styles.css", map[string]string{"Name": "Rita"}, s.customFuncs())

	s.Require().NoError(err)
	s.assertStyledParagraph(got, "custom:Rita")
}

func (s *ManagerSuite) TestRenderHTMLContent() {
	got, err := s.renderManager().RenderHTMLContent("<h1>{{.Title}}</h1>", map[string]string{"Title": "Notice"})

	s.Require().NoError(err)
	s.Equal("<h1>Notice</h1>", got)
}

func (s *ManagerSuite) TestRenderHTMLContentWithFuncs_PassesCustomFuncs() {
	got, err := s.renderManager().RenderHTMLContentWithFuncs("{{toUpper .}}|{{greet .}}", "Rita", s.customFuncs())

	s.Require().NoError(err)
	s.Equal("custom:Rita|hi Rita", got)
}
