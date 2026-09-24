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

	// EmailProviderMailpit captures mail on a local Mailpit server (dev).
	EmailProviderMailpit = "mailpit"
	// EmailProviderResend delivers mail through the Resend API (prod).
	EmailProviderResend = "resend"

	// defaultGenerateCommand produces the UFT glue for one spec file; the
	// spec path is appended by the caller.
	defaultGenerateCommand = "dune exec utf -- gen python"
	// defaultGenerateTimeout bounds one chapter's generation run.
	defaultGenerateTimeout = 5 * time.Minute
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
	Email          EmailConfig
	GitReposDir    string // where the local git server keeps its bare repositories
	Google         GoogleConfig
	Superadmin     SuperadminConfig
	Validation     ValidationConfig
	AssessmentEnv  AssessmentEnvConfig
}

// ValidationConfig controls the push-time checks that need external tooling.
type ValidationConfig struct {
	// GenerateCommand is the shell command that produces the UFT glue for one
	// spec file; the file path is appended and it runs inside the chapter
	// directory. Empty disables the generation gate.
	GenerateCommand string
	// GenerateTimeout bounds one chapter's generation run.
	GenerateTimeout time.Duration
}

// AssessmentEnvConfig holds the runtime defaults for the sandboxed assessment
// runner: the resource limits and timeouts applied to every submission, and the
// paths and image tags the supervisor uses. Everything is environment driven so
// a deployment can tune the sandbox without a rebuild.
type AssessmentEnvConfig struct {
	// SessionTTL is the fallback lifetime of an assessment session whose own
	// config sets no time limit (ASSESSMENT_SESSION_TTL, default 2h).
	SessionTTL time.Duration
	// MemoryBytes is the per-container memory limit in bytes
	// (ASSESSMENT_MEMORY_MB, default 512, converted from MiB).
	MemoryBytes int64
	// NanoCPUs is the per-container CPU quota in units of 1e-9 CPU
	// (ASSESSMENT_NANO_CPUS, default 1000000000, one core).
	NanoCPUs int64
	// PidsLimit caps the number of processes in a container
	// (ASSESSMENT_PIDS_LIMIT, default 256).
	PidsLimit int64
	// BuildTimeout bounds one image build on the sandbox daemon
	// (ASSESSMENT_BUILD_TIMEOUT, default 15m).
	BuildTimeout time.Duration
	// ExecTimeout bounds one command run inside a sandbox container
	// (ASSESSMENT_EXEC_TIMEOUT, default 90s).
	ExecTimeout time.Duration
	// DataDir is the host directory the assessment runtime stores state under
	// (ASSESSMENT_DATA_DIR, default "data").
	DataDir string
	// UTFSourceDir is where the shipped UFT sources live, used to build the
	// harness image on the sandbox daemon (ASSESSMENT_UTF_SRC_DIR, default
	// "/utf-src").
	UTFSourceDir string
	// HarnessImage is the tag of the platform harness builder image
	// (ASSESSMENT_HARNESS_IMAGE, default "codingtest-harness:latest").
	HarnessImage string
	// ImagePrefix prefixes the tags of the per-assessment images
	// (ASSESSMENT_IMAGE_PREFIX, default "codingtest-assessment").
	ImagePrefix string
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

// EmailConfig selects and configures the outbound email transport: Mailpit
// (a local capture server, used in development) or Resend (a real provider,
// used in production). Endpoints only ever see a mailer.Mailer, so the
// choice is invisible to them.
type EmailConfig struct {
	Provider string // EmailProviderMailpit | EmailProviderResend

	// From is the sender displayed in the From header: name + address.
	From string

	// Mailpit is the development transport: plain SMTP to a local server
	// that captures every message and shows it in its web UI.
	MailpitSMTPAddr string // SMTP endpoint, e.g. "mailpit:1025"
	MailpitUIURL    string // web UI base URL, used in log hints

	// ResendAPIKey authenticates the production transport (resend.com).
	ResendAPIKey string
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
	cfg.GitReposDir = get("GIT_REPOS_DIR", "data/git")

	cfg.Validation = ValidationConfig{
		GenerateCommand: generateCommand(),
		GenerateTimeout: getDuration("VALIDATION_GENERATE_TIMEOUT", defaultGenerateTimeout),
	}

	cfg.AssessmentEnv = AssessmentEnvConfig{
		SessionTTL:   getDuration("ASSESSMENT_SESSION_TTL", 2*time.Hour),
		MemoryBytes:  getInt64("ASSESSMENT_MEMORY_MB", 512) << 20, // MiB to bytes
		NanoCPUs:     getInt64("ASSESSMENT_NANO_CPUS", 1_000_000_000),
		PidsLimit:    getInt64("ASSESSMENT_PIDS_LIMIT", 256),
		BuildTimeout: getDuration("ASSESSMENT_BUILD_TIMEOUT", 15*time.Minute),
		ExecTimeout:  getDuration("ASSESSMENT_EXEC_TIMEOUT", 90*time.Second),
		DataDir:      get("ASSESSMENT_DATA_DIR", "data"),
		UTFSourceDir: get("ASSESSMENT_UTF_SRC_DIR", "/utf-src"),
		HarnessImage: get("ASSESSMENT_HARNESS_IMAGE", "codingtest-harness:latest"),
		ImagePrefix:  get("ASSESSMENT_IMAGE_PREFIX", "codingtest-assessment"),
	}

	cfg.Email = EmailConfig{
		Provider:        emailProvider(cfg.Env),
		From:            get("EMAIL_FROM", "QuitLARP <no-reply@example.com>"),
		MailpitSMTPAddr: get("MAILPIT_SMTP_ADDR", "localhost:1025"),
		MailpitUIURL:    strings.TrimRight(get("MAILPIT_UI_URL", "http://localhost:8025"), "/"),
		ResendAPIKey:    os.Getenv("RESEND_API_KEY"),
	}
	if cfg.Env == "production" && cfg.Email.Provider == EmailProviderMailpit {
		log.Printf("WARNING: EMAIL_PROVIDER=mailpit in production — emails are captured, not delivered. Set EMAIL_PROVIDER=resend and RESEND_API_KEY before deploying.")
	}

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

// emailProvider resolves EMAIL_PROVIDER, falling back to the environment:
// development captures mail with Mailpit, production delivers via Resend.
func emailProvider(env string) string {
	if provider := strings.ToLower(get("EMAIL_PROVIDER", "")); provider != "" {
		return provider
	}
	if env == "production" {
		return EmailProviderResend
	}
	return EmailProviderMailpit
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

// generateCommand resolves VALIDATION_GENERATE_COMMAND. Unlike get, an
// explicitly empty value is meaningful here: it disables the generation gate
// so pushes can be validated without a toolchain. Only an unset variable
// falls back to the default.
func generateCommand() string {
	raw, ok := os.LookupEnv("VALIDATION_GENERATE_COMMAND")
	if !ok {
		return defaultGenerateCommand
	}
	return strings.TrimSpace(raw)
}

// getDuration parses a Go duration string such as "5m" or "90s". Values that
// do not parse fall back to the default with a warning instead of failing the
// boot, so a typo degrades one gate rather than the whole server.
func getDuration(key string, fallback time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		log.Printf("WARNING: %s=%q is not a valid duration such as 5m or 90s; using %s", key, raw, fallback)
		return fallback
	}
	return d
}

// getInt64 parses a base-10 integer such as "512". Like getDuration, a value
// that does not parse falls back to the default with a warning instead of
// failing the boot.
func getInt64(key string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	v, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		log.Printf("WARNING: %s=%q is not a valid integer; using %d", key, raw, fallback)
		return fallback
	}
	return v
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
