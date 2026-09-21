// Package config loads all application configuration from environment
// variables. A local `.env` file is picked up when present so secrets never
// have to be hard-coded or committed; values already set in the environment
// (e.g. by docker compose) always win.
package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	defaultJWTSecret     = "dev-only-insecure-jwt-secret-change-me"
	defaultSuperPassword = "superadmin-dev-password-change-me"
)

// Config holds every runtime setting for the backend.
type Config struct {
	Env            string // development | production
	Port           string // HTTP listen address port
	DatabaseURL    string // postgres DSN
	CORSOrigins    []string
	CookieSecure   bool          // set the Secure flag on cookies
	SessionTTL     time.Duration // lifetime of a session JWT / auth cookie
	ActionTokenTTL time.Duration // lifetime of verify-email & password-reset links
	JWTSecret      string
	PublicBaseURL  string // externally reachable base URL of this backend
	FrontendURL    string // where the SPA is served (redirects, email links)
	EmailFrom      string
	GitReposDir    string // where the local git server keeps its bare repositories
	Google         GoogleConfig
	Superadmin     SuperadminConfig
}

// GoogleConfig holds the OAuth2 client credentials for "Sign in with Google".
type GoogleConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURL  string
}

// Enabled reports whether Google OAuth can be used. When the env vars are
// absent the endpoints respond with a clear "not configured" error instead
// of crashing, so the app runs fine without Google credentials.
func (g GoogleConfig) Enabled() bool {
	return g.ClientID != "" && g.ClientSecret != "" && g.RedirectURL != ""
}

// SuperadminConfig describes the single bootstrap super admin for the app.
type SuperadminConfig struct {
	Username string
	Email    string
	Password string
}

// Load reads configuration from the environment plus an optional .env file.
func Load() (*Config, error) {
	// Missing .env file is fine; other load errors are not.
	if err := godotenv.Load(".env"); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("load .env: %w", err)
	}

	cfg := &Config{}

	cfg.Env = get("ENV", "development")
	cfg.Port = get("PORT", "8888")
	cfg.DatabaseURL = os.Getenv("DATABASE_URL")
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required (see backend/.env.example)")
	}

	cfg.CORSOrigins = splitList(get("CORS_ORIGINS", "http://localhost:5173"))
	cfg.CookieSecure = getBool("COOKIE_SECURE", cfg.Env == "production")

	cfg.JWTSecret = get("JWT_SECRET", defaultJWTSecret)
	if cfg.JWTSecret == defaultJWTSecret {
		log.Printf("WARNING: JWT_SECRET not set in the environment; using an insecure development default. Set it in backend/.env before deploying.")
	}

	if hours, err := strconv.Atoi(get("SESSION_TTL_HOURS", "24")); err == nil && hours > 0 {
		cfg.SessionTTL = time.Duration(hours) * time.Hour
	} else {
		cfg.SessionTTL = 24 * time.Hour
	}
	if hours, err := strconv.Atoi(get("ACTION_TOKEN_TTL_HOURS", "24")); err == nil && hours > 0 {
		cfg.ActionTokenTTL = time.Duration(hours) * time.Hour
	} else {
		cfg.ActionTokenTTL = 24 * time.Hour
	}

	cfg.PublicBaseURL = strings.TrimRight(get("PUBLIC_BASE_URL", "http://localhost:8888"), "/")
	cfg.FrontendURL = strings.TrimRight(get("FRONTEND_URL", "http://localhost:5173"), "/")
	cfg.EmailFrom = get("EMAIL_FROM", "QuitLARP <no-reply@example.com>")
	cfg.GitReposDir = get("GIT_REPOS_DIR", "data/git")

	cfg.Google = GoogleConfig{
		ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
		ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		RedirectURL:  get("GOOGLE_REDIRECT_URL", cfg.PublicBaseURL+"/api/v1/auth/google/callback"),
	}

	cfg.Superadmin = SuperadminConfig{
		Username: strings.TrimSpace(get("SUPERADMIN_USERNAME", "superadmin")),
		Email:    strings.TrimSpace(get("SUPERADMIN_EMAIL", "superadmin@example.com")),
		Password: get("SUPERADMIN_PASSWORD", defaultSuperPassword),
	}
	if cfg.Superadmin.Password == defaultSuperPassword {
		log.Printf("WARNING: SUPERADMIN_PASSWORD not set in the environment; using an insecure development default. Set it in backend/.env before deploying.")
	}

	return cfg, nil
}

func get(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getBool(key string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(key))) {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	}
	return fallback
}

func splitList(raw string) []string {
	parts := []string{}
	for _, p := range strings.Split(raw, ",") {
		if p = strings.TrimSpace(p); p != "" {
			parts = append(parts, p)
		}
	}
	return parts
}
