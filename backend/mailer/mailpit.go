package mailer

import (
	"context"
	"fmt"
	"log"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

// Mailpit is the development transport: messages are delivered over plain
// SMTP to a local Mailpit server, which captures them instead of delivering
// them anywhere. Captured mail is browsable in the Mailpit web UI.
type Mailpit struct {
	from  string
	addr  string // SMTP host:port
	uiURL string // web UI base URL, used only for the log hint
}

// NewMailpit creates a Mailpit-backed Mailer.
func NewMailpit(from, addr, uiURL string) *Mailpit {
	return &Mailpit{from: from, addr: addr, uiURL: uiURL}
}

// Send delivers the message to Mailpit over SMTP. A failing connection (for
// example Mailpit not running) is reported to the caller, so a broken dev
// setup surfaces immediately instead of silently dropping mail.
func (m *Mailpit) Send(ctx context.Context, msg Message) error {
	fromAddr := m.from
	if parsed, err := mail.ParseAddress(m.from); err == nil {
		fromAddr = parsed.Address
	}
	host, _, err := net.SplitHostPort(m.addr)
	if err != nil {
		return fmt.Errorf("mailpit: invalid SMTP address %q: %w", m.addr, err)
	}

	conn, err := (&net.Dialer{Timeout: 10 * time.Second}).DialContext(ctx, "tcp", m.addr)
	if err != nil {
		return fmt.Errorf("mailpit: dial %s: %w", m.addr, err)
	}
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		conn.Close()
		return fmt.Errorf("mailpit: connect to %s: %w", m.addr, err)
	}
	defer client.Close()

	if err := client.Mail(fromAddr); err != nil {
		return fmt.Errorf("mailpit: MAIL FROM: %w", err)
	}
	if err := client.Rcpt(msg.To); err != nil {
		return fmt.Errorf("mailpit: RCPT TO: %w", err)
	}
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("mailpit: DATA: %w", err)
	}
	if _, err := w.Write(m.render(msg)); err != nil {
		w.Close()
		return fmt.Errorf("mailpit: write message: %w", err)
	}
	if err := w.Close(); err != nil {
		return fmt.Errorf("mailpit: finish message: %w", err)
	}

	log.Printf("mailpit: captured %q for %s — view it at %s", msg.Subject, msg.To, m.uiURL)
	return client.Quit()
}

// render assembles the RFC 5322 message (headers + plain-text body). Line
// endings are normalized to CRLF for the SMTP wire format.
func (m *Mailpit) render(msg Message) []byte {
	text := strings.ReplaceAll(msg.Text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\n", "\r\n")

	headers := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nDate: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n",
		m.from, msg.To, mime.QEncoding.Encode("utf-8", msg.Subject), time.Now().Format(time.RFC1123Z))
	return []byte(headers + text + "\r\n")
}
