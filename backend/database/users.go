package database

import (
	"context"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

// GetUserByID loads a user by primary key.
func GetUserByID(ctx context.Context, db *sqlx.DB, id int64) (models.User, error) {
	var u models.User
	err := db.GetContext(ctx, &u, "SELECT "+models.UserColumns+" FROM users WHERE id = $1", id)
	if err != nil {
		return models.User{}, mapNotFound(err)
	}
	return u, nil
}

// GetUserByEmail loads a user by case-insensitive email address.
func GetUserByEmail(ctx context.Context, db *sqlx.DB, email string) (models.User, error) {
	var u models.User
	err := db.GetContext(ctx, &u,
		"SELECT "+models.UserColumns+" FROM users WHERE LOWER(email) = LOWER($1)", email)
	if err != nil {
		return models.User{}, mapNotFound(err)
	}
	return u, nil
}

// GetUserWithPassword loads a user plus password hash (email login path).
func GetUserWithPassword(ctx context.Context, db *sqlx.DB, email string) (models.UserWithPassword, error) {
	var row models.UserWithPassword
	err := db.GetContext(ctx, &row,
		"SELECT "+models.UserColumns+", password_hash FROM users WHERE LOWER(email) = LOWER($1)", email)
	if err != nil {
		return models.UserWithPassword{}, mapNotFound(err)
	}
	return row, nil
}

// GetUserWithPasswordByUsername loads a user plus password hash by username
// (the username half of HTTP Basic auth, e.g. for the local git server).
func GetUserWithPasswordByUsername(ctx context.Context, db *sqlx.DB, username string) (models.UserWithPassword, error) {
	var row models.UserWithPassword
	err := db.GetContext(ctx, &row,
		"SELECT "+models.UserColumns+", password_hash FROM users WHERE LOWER(username) = LOWER($1)", username)
	if err != nil {
		return models.UserWithPassword{}, mapNotFound(err)
	}
	return row, nil
}

// GetUserByUsername loads a user by case-insensitive username.
func GetUserByUsername(ctx context.Context, db *sqlx.DB, username string) (models.User, error) {
	var u models.User
	err := db.GetContext(ctx, &u,
		"SELECT "+models.UserColumns+" FROM users WHERE LOWER(username) = LOWER($1)", username)
	if err != nil {
		return models.User{}, mapNotFound(err)
	}
	return u, nil
}

// ListUsers returns every user ordered by id.
func ListUsers(ctx context.Context, db *sqlx.DB) ([]models.User, error) {
	users := []models.User{}
	err := db.SelectContext(ctx, &users,
		"SELECT "+models.UserColumns+" FROM users ORDER BY id")
	return users, err
}

// CreateUser inserts a new user. googleSub and passwordHash may be nil for
// accounts created through the other auth provider.
func CreateUser(ctx context.Context, db *sqlx.DB, username, email string,
	passwordHash *string, role models.Role, emailVerified bool, googleSub *string) (models.User, error) {

	var id int64
	err := db.GetContext(ctx, &id, `
		INSERT INTO users (username, email, password_hash, role, email_verified, google_sub)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id`,
		username, email, passwordHash, role, emailVerified, googleSub)
	if err != nil {
		return models.User{}, err
	}
	return GetUserByID(ctx, db, id)
}

// SetUserRole updates a user's role.
func SetUserRole(ctx context.Context, db *sqlx.DB, id int64, role models.Role) error {
	res, err := db.ExecContext(ctx,
		"UPDATE users SET role = $1, updated_at = NOW() WHERE id = $2", role, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// SetPasswordHash replaces the stored password hash (reset-password flow).
func SetPasswordHash(ctx context.Context, db *sqlx.DB, id int64, hash string) error {
	_, err := db.ExecContext(ctx,
		"UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", hash, id)
	return err
}

// SetEmailVerified toggles the email-verified flag.
func SetEmailVerified(ctx context.Context, db *sqlx.DB, id int64, verified bool) error {
	_, err := db.ExecContext(ctx,
		"UPDATE users SET email_verified = $1, updated_at = NOW() WHERE id = $2", verified, id)
	return err
}

// LinkGoogleByEmail attaches a Google subject id to an existing email account
// and marks the email verified (Google verified it for us).
func LinkGoogleByEmail(ctx context.Context, db *sqlx.DB, email, sub string) error {
	res, err := db.ExecContext(ctx, `
		UPDATE users
		SET google_sub = $1, email_verified = TRUE, updated_at = NOW()
		WHERE LOWER(email) = LOWER($2)`,
		sub, email)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// UsernameTaken reports whether any user already uses the name.
func UsernameTaken(ctx context.Context, db *sqlx.DB, username string) (bool, error) {
	var one int
	err := db.GetContext(ctx, &one,
		"SELECT 1 FROM users WHERE LOWER(username) = LOWER($1) LIMIT 1", username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
