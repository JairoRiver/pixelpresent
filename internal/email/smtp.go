// Package email provides the SMTP-backed domain.EmailSender and an in-memory
// fake for tests.
package email

import (
	"bytes"
	"context"
	"mime"
	"mime/multipart"
	"net"
	"net/smtp"
	"net/textproto"
	"strconv"

	"github.com/JairoRiver/pixelpresent/internal/domain"
)

// SMTPConfig configures an SMTPSender. Username/Password may be empty for
// servers without authentication (e.g. Mailpit in development).
type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

// SMTPSender delivers emails over standard SMTP. A single SendMail call works
// for both Mailpit (no auth, no TLS) and Proton (PlainAuth + automatic STARTTLS).
type SMTPSender struct {
	cfg SMTPConfig
}

var _ domain.EmailSender = (*SMTPSender)(nil)

func NewSMTPSender(cfg SMTPConfig) *SMTPSender {
	return &SMTPSender{cfg: cfg}
}

func (s *SMTPSender) Send(ctx context.Context, msg domain.Email) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	addr := net.JoinHostPort(s.cfg.Host, strconv.Itoa(s.cfg.Port))

	var auth smtp.Auth
	if s.cfg.Username != "" {
		auth = smtp.PlainAuth("", s.cfg.Username, s.cfg.Password, s.cfg.Host)
	}

	return smtp.SendMail(addr, auth, s.cfg.From, []string{msg.To}, buildMessage(s.cfg.From, msg))
}

// buildMessage renders an RFC 5322 message. With both bodies it emits a
// multipart/alternative; otherwise a single text/plain or text/html part.
func buildMessage(from string, msg domain.Email) []byte {
	var buf bytes.Buffer
	writeHeader(&buf, "From", from)
	writeHeader(&buf, "To", msg.To)
	writeHeader(&buf, "Subject", mime.QEncoding.Encode("UTF-8", msg.Subject))
	writeHeader(&buf, "MIME-Version", "1.0")

	hasText := msg.BodyText != ""
	hasHTML := msg.BodyHTML != ""

	switch {
	case hasText && hasHTML:
		mw := multipart.NewWriter(&buf)
		writeHeader(&buf, "Content-Type", "multipart/alternative; boundary="+mw.Boundary())
		buf.WriteString("\r\n")

		textPart, _ := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type": {`text/plain; charset="UTF-8"`},
		})
		_, _ = textPart.Write([]byte(msg.BodyText))

		htmlPart, _ := mw.CreatePart(textproto.MIMEHeader{
			"Content-Type": {`text/html; charset="UTF-8"`},
		})
		_, _ = htmlPart.Write([]byte(msg.BodyHTML))

		_ = mw.Close()
	case hasHTML:
		writeHeader(&buf, "Content-Type", `text/html; charset="UTF-8"`)
		buf.WriteString("\r\n")
		buf.WriteString(msg.BodyHTML)
	default:
		writeHeader(&buf, "Content-Type", `text/plain; charset="UTF-8"`)
		buf.WriteString("\r\n")
		buf.WriteString(msg.BodyText)
	}

	return buf.Bytes()
}

func writeHeader(buf *bytes.Buffer, key, value string) {
	buf.WriteString(key)
	buf.WriteString(": ")
	buf.WriteString(value)
	buf.WriteString("\r\n")
}
