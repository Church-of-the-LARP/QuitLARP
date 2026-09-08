package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Session cookie / header names shared by middleware and handlers.
const (
	SessionCookieName = "auth_token"
	OAuthStateCookie  = "oauth_state"
)

// Token purposes for signed action links (verify email, reset password).
const (
	PurposeVerifyEmail   = "verify-email"
	PurposeResetPassword = "reset-password"
)

// Errors returned while parsing tokens.
var (
	ErrInvalidToken = errors.New("token is invalid or has expired")
)

// Auth signs and verifies the JWTs this app issues. One instance is shared
// by middleware and handlers.
type Auth struct {
	secret         []byte
	sessionTTL     time.Duration
	actionTokenTTL time.Duration
}

// New creates an Auth service.
func New(secret string, sessionTTL, actionTokenTTL time.Duration) *Auth {
	return &Auth{
		secret:         []byte(secret),
		sessionTTL:     sessionTTL,
		actionTokenTTL: actionTokenTTL,
	}
}

// Claims is the payload of a session JWT.
type Claims struct {
	UserID int64 `json:"uid"`
	jwt.RegisteredClaims
}

// SignSession issues a session token for the given user.
func (a *Auth) SignSession(userID int64) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(a.sessionTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "codewhale-auth",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(a.secret)
}

// ParseSession verifies a session token and returns the user id.
func (a *Auth) ParseSession(tokenStr string) (int64, error) {
	var claims Claims
	if _, err := jwt.ParseWithClaims(tokenStr, &claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method %q", t.Method.Alg())
			}
			return a.secret, nil
		},
		jwt.WithIssuer("codewhale-auth"),
	); err != nil {
		return 0, ErrInvalidToken
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		return 0, ErrInvalidToken
	}
	return claims.UserID, nil
}

// SignActionToken issues a short-lived token for a mail-verified action
// (email verification or password reset) bound to one user and purpose.
func (a *Auth) SignActionToken(purpose string, userID int64) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(a.actionTokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        purpose, // reuse the ID claim to carry the purpose
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(a.secret)
}

// ParseActionToken verifies an action token and returns the user id it was
// issued for. expectedPurpose must match the token's purpose.
func (a *Auth) ParseActionToken(expectedPurpose, tokenStr string) (int64, error) {
	var claims Claims
	if _, err := jwt.ParseWithClaims(tokenStr, &claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method %q", t.Method.Alg())
			}
			return a.secret, nil
		},
	); err != nil {
		return 0, ErrInvalidToken
	}
	if claims.ExpiresAt == nil || !claims.ExpiresAt.After(time.Now()) {
		return 0, ErrInvalidToken
	}
	if claims.ID != expectedPurpose {
		return 0, ErrInvalidToken
	}
	return claims.UserID, nil
}
