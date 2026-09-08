package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/oauth2"
	googleoauth "golang.org/x/oauth2/google"
)

// GoogleUser is the subset of the Google userinfo response we consume.
type GoogleUser struct {
	Sub           string `json:"id"`
	Email         string `json:"email"`
	Name          string `json:"name"`
	EmailVerified bool   `json:"verified_email"`
}

// GoogleClient drives the "Sign in with Google" authorization-code flow.
type GoogleClient struct {
	config *oauth2.Config
}

// NewGoogleClient builds a client; it is safe to keep around even when the
// OAuth credentials are empty (Google() then reports not enabled).
func NewGoogleClient(clientID, clientSecret, redirectURL string) *GoogleClient {
	return &GoogleClient{
		config: &oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Scopes:       []string{"openid", "email", "profile"},
			Endpoint:     googleoauth.Endpoint,
		},
	}
}

// Enabled reports whether real credentials are configured.
func (g *GoogleClient) Enabled() bool {
	return g.config.ClientID != "" && g.config.ClientSecret != "" && g.config.RedirectURL != ""
}

// AuthCodeURL builds the Google consent URL for a given CSRF state.
func (g *GoogleClient) AuthCodeURL(state string) string {
	return g.config.AuthCodeURL(state, oauth2.AccessTypeOnline)
}

// Exchange trades the authorization code for an access token.
func (g *GoogleClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return g.config.Exchange(ctx, code)
}

// FetchUser calls the Google userinfo endpoint with the exchanged token.
func (g *GoogleClient) FetchUser(ctx context.Context, tok *oauth2.Token) (*GoogleUser, error) {
	client := g.config.Client(ctx, tok)
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://www.googleapis.com/oauth2/v2/userinfo", nil)
	if err != nil {
		return nil, fmt.Errorf("google userinfo request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("google userinfo: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("google userinfo returned status %d", resp.StatusCode)
	}
	var user GoogleUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("decode google userinfo: %w", err)
	}
	if user.Sub == "" || user.Email == "" {
		return nil, fmt.Errorf("google userinfo missing id or email")
	}
	return &user, nil
}
