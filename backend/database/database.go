// Package database owns the postgres connection, schema migrations and the
// row-level access functions used by handlers and middleware.
package database

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib" // register the "pgx" driver
	"github.com/jmoiron/sqlx"
)

// ErrNotFound is returned when a lookup matched no rows.
var ErrNotFound = errors.New("not found")

// Connect dials the database, retrying while the container orchestration
// brings postgres up.
func Connect(dsn string, retries int, delay time.Duration) (*sqlx.DB, error) {
	var db *sqlx.DB
	var err error
	for range retries {
		db, err = sqlx.Connect("pgx", dsn)
		if err == nil {
			return db, nil
		}
		log.Printf("database not ready: %v", err)
		time.Sleep(delay)
	}
	return nil, fmt.Errorf("could not connect to database after %d tries: %w", retries, err)
}

// IsUniqueViolation reports whether err is a postgres unique-constraint
// violation (SQLSTATE 23505), e.g. duplicate username/email.
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
