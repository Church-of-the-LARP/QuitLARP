package database

import (
	"context"
	"embed"
	"log"

	"github.com/jmoiron/sqlx"
	"github.com/pressly/goose/v3"

	"backend/auth"
	"backend/models"
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

// SeedAdmin guarantees the configured bootstrap admin account exists. It
// creates the account with the admin role on first boot and re-asserts that role
// if it has drifted. Other users' roles are never touched; the password is only
// set on creation.
func SeedAdmin(ctx context.Context, db *sqlx.DB, username, email, password string) error {
	existing, err := GetUserByUsername(ctx, db, username)
	switch {
	case err == nil:
		// The bootstrap account must always be an admin so a bad role change
		// is recoverable with a restart; also synchronize identity fields.
		if existing.Role != models.RoleAdmin {
			if err := SetUserRole(ctx, db, existing.ID, models.RoleAdmin); err != nil {
				return err
			}
			log.Printf("bootstrap admin account %q role restored to admin", username)
		}
		if !existing.EmailVerified || existing.Email != email {
			if _, err := db.ExecContext(ctx, `
				UPDATE users
				SET email_verified = TRUE, email = $1, updated_at = NOW()
				WHERE id = $2`, email, existing.ID); err != nil {
				return err
			}
			log.Printf("bootstrap admin account %q email/verification synced from env", username)
		}
		return nil
	case err == ErrNotFound:
		hash, err := auth.HashPassword(password)
		if err != nil {
			return err
		}
		if _, err := CreateUser(ctx, db, username, email, &hash, "admin", true, nil); err != nil {
			return err
		}
		log.Printf("created bootstrap admin account %q from environment config", username)
		return nil
	default:
		return err
	}
}
