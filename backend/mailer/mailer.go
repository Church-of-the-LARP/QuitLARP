// Package mailer sends outbound application emails. It only contains a mock
// implementation: every message is written to the backend log so flows such
// as email verification and password reset can be exercised end-to-end
// without a real email provider.
package mailer

import (
	"context"
	"fmt"
	"log"
)

// Message is a single outgoing email.
type Message struct {
	To      string
	Subject string
	Text    string
}

// Mailer delivers messages to recipients.
type Mailer interface {
	Send(ctx context.Context, msg Message) error
}

// Mock is a Mailer that prints messages to the console. Replace it with a
// real SMTP/provider-backed implementation later without touching callers.
type Mock struct {
	From string
}

// NewMock creates a console Mailer.
func NewMock(from string) *Mock {
	return &Mock{From: from}
}

// Send logs the message. The full "email" is visible with:
//
//	docker compose logs backend
func (m *Mock) Send(_ context.Context, msg Message) error {
	body := fmt.Sprintf(`
--------------------------------- MOCK EMAIL ---------------------------------
From:    %s
To:      %s
Subject: %s

%s
--------------------------------- END EMAIL ---------------------------------
`, m.From, msg.To, msg.Subject, msg.Text)
	log.Print(body)
	return nil
}
