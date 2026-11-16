package mailstyler

import (
	"encoding/base64"
	"net"
	"net/textproto"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// fakeSMTPOptions controls how the fake server responds
type fakeSMTPOptions struct {
	advertiseAuth bool // announce "AUTH PLAIN" in the EHLO response
	rejectRcpt    bool // answer every RCPT with 550
}

// fakeSMTPRecord holds everything the server received during a session
type fakeSMTPRecord struct {
	mailFrom   string
	rcpts      []string
	authCalled bool
	authUser   string
	authPass   string
	data       string // DATA payload with dot-stuffing removed and CRLF normalized to LF
}

// fakeSMTPServer is a minimal in-process SMTP server for SendMail tests.
// It never announces STARTTLS, so net/smtp talks to it in plain text.
type fakeSMTPServer struct {
	listener net.Listener
	opts     fakeSMTPOptions

	mu     sync.Mutex
	record fakeSMTPRecord
}

func newFakeSMTPServer(t *testing.T, opts fakeSMTPOptions) *fakeSMTPServer {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to start fake SMTP server: %v", err)
	}

	s := &fakeSMTPServer{listener: listener, opts: opts}
	t.Cleanup(func() { listener.Close() })

	go s.serve()

	return s
}

// Port returns the port the server listens on
func (s *fakeSMTPServer) Port() string {
	return strconv.Itoa(s.listener.Addr().(*net.TCPAddr).Port)
}

// Record returns a copy of what the server has received so far
func (s *fakeSMTPServer) Record() fakeSMTPRecord {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec := s.record
	rec.rcpts = append([]string(nil), s.record.rcpts...)
	return rec
}

func (s *fakeSMTPServer) serve() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			return
		}
		go s.handle(conn)
	}
}

func (s *fakeSMTPServer) handle(conn net.Conn) {
	tp := textproto.NewConn(conn)
	defer tp.Close()

	tp.PrintfLine("220 fake ESMTP ready")

	for {
		line, err := tp.ReadLine()
		if err != nil {
			return
		}

		verb, arg, _ := strings.Cut(line, " ")

		switch strings.ToUpper(verb) {
		case "EHLO":
			if s.opts.advertiseAuth {
				tp.PrintfLine("250-fake")
				tp.PrintfLine("250 AUTH PLAIN")
			} else {
				tp.PrintfLine("250 fake")
			}
		case "HELO", "NOOP", "RSET":
			tp.PrintfLine("250 OK")
		case "AUTH":
			s.handleAuth(tp, arg)
		case "MAIL":
			s.mu.Lock()
			s.record.mailFrom = extractAddress(arg)
			s.mu.Unlock()
			tp.PrintfLine("250 OK")
		case "RCPT":
			if s.opts.rejectRcpt {
				tp.PrintfLine("550 5.1.1 mailbox unavailable")
				continue
			}
			s.mu.Lock()
			s.record.rcpts = append(s.record.rcpts, extractAddress(arg))
			s.mu.Unlock()
			tp.PrintfLine("250 OK")
		case "DATA":
			tp.PrintfLine("354 end data with <CR><LF>.<CR><LF>")
			data, err := tp.ReadDotBytes()
			if err != nil {
				return
			}
			s.mu.Lock()
			s.record.data = string(data)
			s.mu.Unlock()
			tp.PrintfLine("250 OK queued")
		case "QUIT":
			tp.PrintfLine("221 bye")
			return
		default:
			tp.PrintfLine("502 command not implemented")
		}
	}
}

// handleAuth accepts "AUTH PLAIN <base64(\x00user\x00pass)>"
func (s *fakeSMTPServer) handleAuth(tp *textproto.Conn, arg string) {
	mech, resp, _ := strings.Cut(arg, " ")
	if !strings.EqualFold(mech, "PLAIN") {
		tp.PrintfLine("504 unrecognized authentication type")
		return
	}

	decoded, err := base64.StdEncoding.DecodeString(resp)
	if err != nil {
		tp.PrintfLine("501 invalid base64")
		return
	}

	parts := strings.Split(string(decoded), "\x00")
	if len(parts) != 3 {
		tp.PrintfLine("501 malformed PLAIN response")
		return
	}

	s.mu.Lock()
	s.record.authCalled = true
	s.record.authUser = parts[1]
	s.record.authPass = parts[2]
	s.mu.Unlock()

	tp.PrintfLine("235 2.7.0 authentication successful")
}

// extractAddress turns "FROM:<a@b.c> BODY=8BITMIME" into "a@b.c"
func extractAddress(arg string) string {
	start := strings.Index(arg, "<")
	end := strings.Index(arg, ">")
	if start < 0 || end < start {
		return ""
	}
	return arg[start+1 : end]
}
