// Package auth contains everything security related that is not HTTP
// middleware: password hashing/verification, signed session tokens (JWTs)
// and the Google OAuth2 client.
package auth

import (
	"errors"
	"fmt"
	"regexp"

	"golang.org/x/crypto/bcrypt"
)

var usernameRe = regexp.MustCompile(`^[A-Za-z0-9_-]{3,32}$`)

// Password errors surfaced to the API layer.
var (
	ErrPasswordTooShort = errors.New("password must be at least 8 characters")
	ErrPasswordTooLong  = errors.New("password must be at most 72 characters")
	ErrInvalidUsername  = errors.New("username must be 3-32 characters using letters, digits, '-' or '_'")
	ErrInvalidEmail     = errors.New("a valid email address is required")
)

const (
	minPasswordLen = 8
	maxPasswordLen = 72 // bcrypt input limit
)

// ValidatePassword enforces the shared password policy.
func ValidatePassword(password string) error {
	if len(password) < minPasswordLen {
		return ErrPasswordTooShort
	}
	if len(password) > maxPasswordLen {
		return ErrPasswordTooLong
	}
	return nil
}

// ValidateUsername checks the allowed character set and length.
func ValidateUsername(username string) error {
	if !usernameRe.MatchString(username) {
		return ErrInvalidUsername
	}
	return nil
}

// HashPassword returns a bcrypt hash. Bcrypt embeds a random salt in the
// hash itself, so every call produces a distinct value.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword reports whether password matches the stored bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
