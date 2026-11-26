// Package mailer sends outbound application emails. The transport is chosen
// centrally from configuration (see config.EmailConfig): development captures
// every message with a local Mailpit server, production delivers through
// Resend. Endpoints only ever see the Mailer interface, so they never need
// to know which transport is active.
package mailer

import (
	"context"
	"fmt"

	"backend/config"
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

// New builds the Mailer selected by cfg.Provider. It fails fast on unknown
// providers and on a Resend configuration without an API key, so a
// misconfigured service refuses to start instead of dropping mail later.
func New(cfg config.EmailConfig) (Mailer, error) {
	switch cfg.Provider {
	case config.EmailProviderMailpit:
		return NewMailpit(cfg.From, cfg.MailpitSMTPAddr, cfg.MailpitUIURL), nil
	case config.EmailProviderResend:
		if cfg.ResendAPIKey == "" {
			return nil, fmt.Errorf("EMAIL_PROVIDER=%s requires RESEND_API_KEY (see backend/.env.example)", config.EmailProviderResend)
		}
		return NewResend(cfg.From, cfg.ResendAPIKey), nil
	default:
		return nil, fmt.Errorf("unknown EMAIL_PROVIDER %q (expected %q or %q)", cfg.Provider, config.EmailProviderMailpit, config.EmailProviderResend)
	}
}
