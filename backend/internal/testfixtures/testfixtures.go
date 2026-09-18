// Package testfixtures holds the database test helpers shared by the
// database and runtime packages' test suites, so a schema or fixture change
// only has to be made in one place.
package testfixtures

import (
	"context"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"

	"backend/database"
	"backend/models"
)

// DB connects to a real postgres instance (see compose.yaml's `db` service)
// and runs migrations against it. Invite-acceptance and runner-phase
// correctness genuinely depend on Postgres semantics (row locking, NOW()),
// so callers skip rather than mock when no database is reachable.
func DB(t *testing.T) *sqlx.DB {
	t.Helper()
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://app:app@localhost:5432/app?sslmode=disable"
	}
	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		t.Skipf("no test database reachable at %q: %v", dsn, err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("test database not reachable: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

var seq atomic.Int64

// Unique includes the process id so two test binaries running against the
// same shared database concurrently (e.g. `go test ./...` running the
// database and runtime packages' tests at once) can never collide, even if
// they land in the same timestamp tick.
func Unique(prefix string) string {
	n := seq.Add(1)
	return prefix + "-" + strconv.Itoa(os.Getpid()) + "-" + time.Now().Format("150405.000000") + "-" + strconv.FormatInt(n, 10)
}

// User inserts a minimal user row and returns its id.
func User(t *testing.T, db *sqlx.DB) int64 {
	t.Helper()
	name := Unique("user")
	var id int64
	err := db.Get(&id, `
		INSERT INTO users (username, email, password_hash, email_verified)
		VALUES ($1, $2, 'x', true) RETURNING id`, name, name+"@example.com")
	if err != nil {
		t.Fatalf("insert user: %v", err)
	}
	return id
}

// Assessment inserts a minimal assessment row and returns its id.
func Assessment(t *testing.T, db *sqlx.DB, authorID int64, timeLimitMinutes int) int64 {
	t.Helper()
	var id int64
	err := db.Get(&id, `
		INSERT INTO assessments (title, description, difficulty, time_limit_minutes, template_file_name, author_id)
		VALUES ($1, 'desc', 'easy', $2, 'main.go', $3) RETURNING id`,
		Unique("assessment"), timeLimitMinutes, authorID)
	if err != nil {
		t.Fatalf("insert assessment: %v", err)
	}
	return id
}

// Chapters inserts n chapters (positions 1..n) for an assessment and returns
// their ids in order.
func Chapters(t *testing.T, db *sqlx.DB, assessmentID int64, n int) []int64 {
	t.Helper()
	ids := make([]int64, n)
	for i := range n {
		var id int64
		err := db.Get(&id, `
			INSERT INTO chapters (assessment_id, position, title, description, time_limit_minutes)
			VALUES ($1, $2, $3, 'desc', 10) RETURNING id`,
			assessmentID, i+1, Unique("chapter"))
		if err != nil {
			t.Fatalf("insert chapter: %v", err)
		}
		ids[i] = id
	}
	return ids
}

// Attempt inserts an attempt row with full control over fields the public
// database.CreateAttempt doesn't expose (inviteId, startedAt), so tests can
// construct "already running" / "already scheduled" fixtures directly,
// exactly as a Runner would find them after a restart.
func Attempt(t *testing.T, db *sqlx.DB, assessmentID, userID int64, inviteID *int64, timeLimitMinutes int, startedAt *time.Time) int64 {
	t.Helper()
	status := models.AttemptPending
	if startedAt != nil {
		status = models.AttemptRunning
	}
	var id int64
	err := db.Get(&id, `
		INSERT INTO assessment_attempts (assessment_id, invite_id, user_id, status, time_limit_minutes, started_at)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		assessmentID, inviteID, userID, status, timeLimitMinutes, startedAt)
	if err != nil {
		t.Fatalf("insert attempt: %v", err)
	}
	return id
}

// Invite inserts an invite via the real database.CreateInvite, so its
// invite_code generation is exercised too.
func Invite(t *testing.T, db *sqlx.DB, assessmentID, hostID int64, scheduledStartAt time.Time, timeLimitMinutes, maxUses *int) models.Invite {
	t.Helper()
	invite, err := database.CreateInvite(context.Background(), db, assessmentID, hostID, scheduledStartAt, timeLimitMinutes, maxUses)
	if err != nil {
		t.Fatalf("create invite: %v", err)
	}
	return invite
}
