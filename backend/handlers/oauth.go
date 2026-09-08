package handlers

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"backend/auth"
	"backend/database"
	"backend/models"
)

// GoogleAuthStart begins the "Sign in with Google" flow: it stores a CSRF
// state cookie and redirects the browser to Google's consent screen.
// Implemented as a raw http handler because it is a browser redirect, not a
// JSON API operation.
func (h *Handlers) GoogleAuthStart(w http.ResponseWriter, r *http.Request) {
	if !h.google.Enabled() {
		writeProblem(w, http.StatusNotImplemented,
			"Google OAuth is not configured: set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET (see backend/.env.example)")
		return
	}
	state, err := randomHex(16)
	if err != nil {
		writeProblem(w, http.StatusInternalServerError, "could not start sign-in")
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     auth.OAuthStateCookie,
		Value:    state,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   600, // 10 minutes to finish the flow
	})
	http.Redirect(w, r, h.google.AuthCodeURL(state), http.StatusFound)
}

// GoogleAuthCallback receives Google's redirect with the authorization code,
// exchanges it, finds-or-creates the local user and signs them in.
func (h *Handlers) GoogleAuthCallback(w http.ResponseWriter, r *http.Request) {
	fail := func(msg string) {
		h.clearOAuthState(w)
		http.Redirect(w, r, h.cfg.FrontendURL+"?error="+url.QueryEscape(msg), http.StatusFound)
	}

	if !h.google.Enabled() {
		writeProblem(w, http.StatusNotImplemented,
			"Google OAuth is not configured: set GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET (see backend/.env.example)")
		return
	}
	if denied := r.URL.Query().Get("error"); denied != "" {
		fail("google sign-in was cancelled")
		return
	}
	code := r.URL.Query().Get("code")
	if code == "" {
		fail("google sign-in failed: no authorization code received")
		return
	}
	// CSRF: the state we get back must match the cookie we set.
	state := r.URL.Query().Get("state")
	cookie, err := r.Cookie(auth.OAuthStateCookie)
	if err != nil || state == "" ||
		subtle.ConstantTimeCompare([]byte(state), []byte(cookie.Value)) != 1 {
		fail("sign-in failed: state mismatch, please try again")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
	defer cancel()

	tok, err := h.google.Exchange(ctx, code)
	if err != nil {
		log.Printf("google exchange: %v", err)
		fail("sign-in failed: could not contact google")
		return
	}
	guser, err := h.google.FetchUser(ctx, tok)
	if err != nil {
		log.Printf("google userinfo: %v", err)
		fail("sign-in failed: could not fetch your google profile")
		return
	}

	u, err := h.googleUpsert(ctx, guser)
	if err != nil {
		if database.IsUniqueViolation(err) {
			log.Printf("google upsert conflict: %v", err)
			fail("sign-in failed: this email is already linked to another google account")
			return
		}
		log.Printf("google upsert: %v", err)
		fail("sign-in failed: could not create or update your account")
		return
	}
	if err := h.signIn(w, u); err != nil {
		log.Printf("google sign in token: %v", err)
		fail("sign-in failed: could not create a session")
		return
	}
	h.clearOAuthState(w)
	http.Redirect(w, r, h.cfg.FrontendURL, http.StatusFound)
}

// googleUpsert links a verified Google profile to an existing account with
// the same email or creates a new user (role: user, email pre-verified).
func (h *Handlers) googleUpsert(ctx context.Context, g *auth.GoogleUser) (models.User, error) {
	email := strings.ToLower(strings.TrimSpace(g.Email))

	if _, err := database.GetUserByEmail(ctx, h.db, email); err == nil {
		// Link this Google identity to the existing account.
		if err := database.LinkGoogleByEmail(ctx, h.db, email, g.Sub); err != nil {
			return models.User{}, err
		}
		return database.GetUserByEmail(ctx, h.db, email)
	} else if !errors.Is(err, database.ErrNotFound) {
		return models.User{}, err
	}

	username, err := h.uniqueUsername(ctx, googleUsername(g))
	if err != nil {
		return models.User{}, err
	}
	sub := g.Sub
	return database.CreateUser(ctx, h.db, username, email, nil, models.RoleUser, true, &sub)
}

// uniqueUsername finds a free name close to base ("alice", "alice2", ...).
func (h *Handlers) uniqueUsername(ctx context.Context, base string) (string, error) {
	base = cleanUsername(base)
	for i := 1; i <= 100; i++ {
		candidate := base
		if i > 1 {
			candidate = fmt.Sprintf("%.20s%d", base, i)
		}
		taken, err := database.UsernameTaken(ctx, h.db, candidate)
		if err != nil {
			return "", err
		}
		if !taken {
			return candidate, nil
		}
	}
	return "", errors.New("could not allocate a unique username")
}

// googleUsername prefers the profile name, falls back to the email prefix.
func googleUsername(g *auth.GoogleUser) string {
	if g.Name != "" {
		return g.Name
	}
	if at := strings.IndexByte(g.Email, '@'); at > 0 {
		return g.Email[:at]
	}
	return g.Email
}

// cleanUsername keeps only [A-Za-z0-9_-] runes and guarantees 3-20 chars.
func cleanUsername(raw string) string {
	var b strings.Builder
	for _, r := range raw {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '_':
			b.WriteRune(r)
		}
	}
	out := strings.Trim(b.String(), "_-")
	if out == "" {
		out = "user"
	}
	if runes := []rune(out); len(runes) < 3 {
		out = "user" + out
	} else if len(runes) > 20 {
		out = string(runes[:20])
	}
	return out
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func (h *Handlers) clearOAuthState(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     auth.OAuthStateCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
