// Package handlers wires the HTTP surface of the app. Huma-style operations
// (typed, OpenAPI-documented JSON endpoints) live next to the raw http
// handlers needed for browser redirect flows (Google OAuth).
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/mail"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/jmoiron/sqlx"

	"backend/auth"
	"backend/config"
	"backend/database"
	"backend/mailer"
	"backend/middleware"
	"backend/models"
	"backend/runtime"
)

// Handlers carries the shared dependencies for every endpoint.
type Handlers struct {
	db       *sqlx.DB
	cfg      *config.Config
	mail     mailer.Mailer
	tokens   *auth.Auth
	google   *auth.GoogleClient
	attempts *runtime.AttemptManager
}

// New builds the handler bundle.
func New(db *sqlx.DB, cfg *config.Config, mail mailer.Mailer,
	tokens *auth.Auth, google *auth.GoogleClient, attempts *runtime.AttemptManager) *Handlers {
	return &Handlers{db: db, cfg: cfg, mail: mail, tokens: tokens, google: google, attempts: attempts}
}

// Register mounts every huma operation on the API.
func (h *Handlers) Register(api huma.API) {
	h.registerHealth(api)
	h.registerAuth(api)
	h.registerEmailFlows(api)
	h.registerUsers(api)
	h.registerAssessments(api)
	h.registerAssessmentChapters(api)
	h.registerAssessmentTests(api)
	h.registerAssessmentInvites(api)
	h.registerAssessmentAttempts(api)
	h.registerTags(api)
}

// validEmail does a light syntax check (format only; ownership is proven by
// the verification email flow).
func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return err == nil && addr.Address == email && strings.Contains(email, ".")
}

// requireUser resolves the current user or maps auth failures to huma errors.
// With no roles passed any authenticated user passes; with roles, the user's
// role must be one of them.
func (h *Handlers) requireUser(ctx context.Context, allowed ...models.Role) (models.User, error) {
	var u models.User
	var err error
	if len(allowed) == 0 {
		u, err = middleware.CurrentUser(ctx, h.db)
	} else {
		u, err = middleware.RequireAnyRole(ctx, h.db, allowed...)
	}
	if err == nil {
		return u, nil
	}
	return models.User{}, toHTTPError(err)
}

func toHTTPError(err error) error {
	switch {
	case errors.Is(err, middleware.ErrUnauthenticated):
		return huma.NewError(http.StatusUnauthorized, "authentication required — please sign in")
	case errors.Is(err, middleware.ErrForbidden):
		return huma.NewError(http.StatusForbidden, "you do not have permission to do this")
	case errors.Is(err, database.ErrNotFound):
		return huma.NewError(http.StatusNotFound, "not found")
	case database.IsUniqueViolation(err):
		return huma.NewError(http.StatusConflict, "that email or username is already registered")
	default:
		log.Printf("internal error: %v", err)
		return huma.NewError(http.StatusInternalServerError, "something went wrong")
	}
}

// sessionCookie builds the auth cookie for the given token value.
func (h *Handlers) sessionCookie(value string, maxAge int) *http.Cookie {
	return &http.Cookie{
		Name:     auth.SessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.CookieSecure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	}
}

// signIn attaches a fresh session cookie for the user (raw HTTP path, used by
// the Google callback which is not a huma operation).
func (h *Handlers) signIn(w http.ResponseWriter, u models.User) error {
	token, err := h.tokens.SignSession(u.ID)
	if err != nil {
		return err
	}
	http.SetCookie(w, h.sessionCookie(token, int(h.cfg.SessionTTL.Seconds())))
	return nil
}

// raw problem-style JSON responses for the non-huma endpoints.
type problem struct {
	Detail string `json:"detail"`
}

func writeProblem(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem{Detail: detail})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// devMail is the mock delivery note appended to every "email": the message
// itself already lands in the backend log.
const devMail = "\n(dev: this is a mock email — it was printed to the backend log, not sent. Run `docker compose logs backend` to read it.)"

// sendVerificationEmail issues a signed link and hands it to the mailer.
func (h *Handlers) sendVerificationEmail(ctx context.Context, u models.User) error {
	if u.EmailVerified {
		return nil
	}
	token, err := h.tokens.SignActionToken(auth.PurposeVerifyEmail, u.ID)
	if err != nil {
		return err
	}
	link := h.cfg.PublicBaseURL + "/api/v1/auth/verify-email?token=" + token
	return h.mail.Send(ctx, mailer.Message{
		To:      u.Email,
		Subject: "Verify your email address",
		Text:    "Hi " + u.Username + ",\n\nPlease confirm your email address by opening this link:\n\n" + link + "\n\nThe link is valid for " + h.cfg.ActionTokenTTL.String() + ". If you did not create an account, you can ignore this email." + devMail,
	})
}

// sendResetEmail issues a signed password-reset link pointed at the SPA.
func (h *Handlers) sendResetEmail(ctx context.Context, u models.User) error {
	token, err := h.tokens.SignActionToken(auth.PurposeResetPassword, u.ID)
	if err != nil {
		return err
	}
	link := h.cfg.FrontendURL + "/reset-password?token=" + token
	return h.mail.Send(ctx, mailer.Message{
		To:      u.Email,
		Subject: "Reset your password",
		Text:    "Hi " + u.Username + ",\n\nSomeone asked to reset your password. Open this link to choose a new one:\n\n" + link + "\n\nThe link is valid for " + h.cfg.ActionTokenTTL.String() + ". If this was not you, you can ignore this email." + devMail,
	})
}
