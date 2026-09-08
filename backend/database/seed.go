package database

import (
	"context"
	"embed"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"

	"backend/auth"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Migrate applies pending schema migrations.
func Migrate(db *sqlx.DB) error {
	goose.SetBaseFS(migrations)
	goose.SetDialect("postgres")
	if err := goose.Up(db.DB, "migrations"); err != nil {
		return err
	}
	return nil
}

// SeedSuperadmin guarantees the single bootstrap super admin from the
// configuration exists. It creates the account on first boot and afterwards
// only repairs drift (wrong role, unverified email). The password is only
// set on creation; later changes go through the normal reset flow.
func SeedSuperadmin(ctx context.Context, db *sqlx.DB, username, email, password string) error {
	existing, err := GetUserByUsername(ctx, db, username)
	switch {
	case err == nil:
		// Account exists: keep role + email verification in sync with config.
		if existing.Role != "superadmin" || !existing.EmailVerified || existing.Email != email {
			if _, err := db.ExecContext(ctx, `
				UPDATE users
				SET role = 'superadmin', email_verified = TRUE, email = $1, updated_at = NOW()
				WHERE id = $2`, email, existing.ID); err != nil {
				return err
			}
			log.Printf("superadmin account %q repaired (role/email/verification synced from env)", username)
		}
		return nil
	case err == ErrNotFound:
		hash, err := auth.HashPassword(password)
		if err != nil {
			return err
		}
		if _, err := CreateUser(ctx, db, username, email, &hash, "superadmin", true, nil); err != nil {
			return err
		}
		log.Printf("created superadmin account %q from environment config", username)
		return nil
	default:
		return err
	}
}
