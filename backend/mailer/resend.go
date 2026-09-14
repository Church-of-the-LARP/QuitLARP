package mailer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// resendEndpoint is Resend's "send email" API.
// See https://resend.com/docs/api-reference/emails/send-email.
const resendEndpoint = "https://api.resend.com/emails"

// Resend is the production transport: messages are delivered through the
// Resend API using an API key.
type Resend struct {
	from   string
	apiKey string
	client *http.Client
}

// NewResend creates a Resend-backed Mailer.
func NewResend(from, apiKey string) *Resend {
	return &Resend{
		from:   from,
		apiKey: apiKey,
		client: &http.Client{Timeout: 15 * time.Second},
	}
}

// Send posts the message to the Resend API. Delivery errors carry the
// provider's response body, so misconfiguration (bad key, unverified domain)
// is visible in the backend log instead of failing silently.
func (r *Resend) Send(ctx context.Context, msg Message) error {
	payload, err := json.Marshal(map[string]any{
		"from":    r.from,
		"to":      []string{msg.To},
		"subject": msg.Subject,
		"text":    msg.Text,
	})
	if err != nil {
		return fmt.Errorf("resend: encode request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, resendEndpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("resend: build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+r.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.client.Do(req)
	if err != nil {
		return fmt.Errorf("resend: send: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		detail, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("resend: %s: %s", resp.Status, strings.TrimSpace(string(detail)))
	}
	return nil
}
