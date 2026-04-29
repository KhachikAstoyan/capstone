package email

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"mime"
	"mime/quotedprintable"
	"net/mail"
	"net/smtp"
	"time"
)

const verificationTmplSrc = `<!DOCTYPE html>
<html lang="en">
<head><meta charset="utf-8"><meta name="viewport" content="width=device-width,initial-scale=1"></head>
<body style="font-family:system-ui,sans-serif;background:#f9fafb;margin:0;padding:32px 16px">
  <div style="max-width:560px;margin:0 auto;background:#fff;border-radius:8px;padding:40px;box-shadow:0 1px 3px rgba(0,0,0,.1)">
    <h1 style="margin:0 0 8px;font-size:22px;color:#111">Verify your email address</h1>
    <p style="color:#374151;margin:0 0 24px">Click the button below to verify your email and activate your account.</p>
    <a href="{{.URL}}"
       style="display:inline-block;padding:12px 28px;background:#2563eb;color:#fff;text-decoration:none;border-radius:6px;font-weight:600">
      Verify Email
    </a>
    <p style="margin:24px 0 8px;color:#6b7280;font-size:13px">Or copy this link into your browser:</p>
    <p style="margin:0;word-break:break-all;font-size:13px">
      <a href="{{.URL}}" style="color:#2563eb">{{.URL}}</a>
    </p>
    <hr style="margin:32px 0;border:none;border-top:1px solid #e5e7eb">
    <p style="margin:0;color:#9ca3af;font-size:12px">This link expires in 48 hours. If you did not create an account, you can safely ignore this email.</p>
  </div>
</body>
</html>`

var verificationTmpl = template.Must(template.New("verification").Parse(verificationTmplSrc))

// Sender sends transactional emails over SMTP.
type Sender struct {
	host     string
	port     int
	username string
	password string
	from     string
}

// NewSender constructs a Sender from config.
func NewSender(cfg *Config) *Sender {
	return &Sender{
		host:     cfg.SMTPHost,
		port:     cfg.SMTPPort,
		username: cfg.SMTPUsername,
		password: cfg.SMTPPassword,
		from:     cfg.SMTPFrom,
	}
}

// SendVerificationEmail sends the email verification link to toEmail.
func (s *Sender) SendVerificationEmail(toEmail, verificationURL string) error {
	var buf bytes.Buffer
	if err := verificationTmpl.Execute(&buf, struct{ URL string }{URL: verificationURL}); err != nil {
		return fmt.Errorf("render verification template: %w", err)
	}
	return s.sendHTML(toEmail, "Verify your email address", buf.String())
}

func (s *Sender) sendHTML(to, subject, html string) error {
	fromAddr := mail.Address{Address: s.from}
	toAddr := mail.Address{Address: to}

	var raw bytes.Buffer
	fmt.Fprintf(&raw, "MIME-Version: 1.0\r\n")
	fmt.Fprintf(&raw, "Date: %s\r\n", time.Now().UTC().Format(time.RFC1123Z))
	fmt.Fprintf(&raw, "From: %s\r\n", fromAddr.String())
	fmt.Fprintf(&raw, "To: %s\r\n", toAddr.String())
	fmt.Fprintf(&raw, "Subject: %s\r\n", mime.QEncoding.Encode("utf-8", subject))
	fmt.Fprintf(&raw, "Content-Type: text/html; charset=UTF-8\r\n")
	fmt.Fprintf(&raw, "Content-Transfer-Encoding: quoted-printable\r\n")
	fmt.Fprintf(&raw, "\r\n")

	qw := quotedprintable.NewWriter(&raw)
	if _, err := qw.Write([]byte(html)); err != nil {
		return fmt.Errorf("encode body: %w", err)
	}
	if err := qw.Close(); err != nil {
		return fmt.Errorf("flush body: %w", err)
	}

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	auth := smtp.PlainAuth("", s.username, s.password, s.host)

	// Port 465 uses implicit TLS; all others use STARTTLS via smtp.SendMail.
	if s.port == 465 {
		return s.sendImplicitTLS(addr, auth, to, raw.Bytes())
	}
	return smtp.SendMail(addr, auth, s.from, []string{to}, raw.Bytes())
}

func (s *Sender) sendImplicitTLS(addr string, auth smtp.Auth, to string, msg []byte) error {
	conn, err := tls.Dial("tcp", addr, &tls.Config{ServerName: s.host})
	if err != nil {
		return fmt.Errorf("tls dial: %w", err)
	}
	client, err := smtp.NewClient(conn, s.host)
	if err != nil {
		return fmt.Errorf("smtp client: %w", err)
	}
	defer client.Close()

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("smtp auth: %w", err)
	}
	if err := client.Mail(s.from); err != nil {
		return fmt.Errorf("smtp MAIL FROM: %w", err)
	}
	if err := client.Rcpt(to); err != nil {
		return fmt.Errorf("smtp RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("smtp DATA: %w", err)
	}
	if _, err := w.Write(msg); err != nil {
		return fmt.Errorf("smtp write body: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("smtp close data: %w", err)
	}
	return client.Quit()
}
