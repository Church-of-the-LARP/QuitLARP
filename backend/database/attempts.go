package database

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

// attemptColumns is the SELECT list for a full attempt row.
const attemptColumns = `id, assessment_id, invite_id, user_id, status, time_limit_minutes,
	current_chapter_id, started_at, ended_at`

// CreateAttempt inserts a solo, self-started attempt (inviteId NULL).
func CreateAttempt(ctx context.Context, db *sqlx.DB, assessmentID, userID int64, timeLimitMinutes int) (models.Attempt, error) {
	var attempt models.Attempt
	err := db.GetContext(ctx, &attempt, `
		INSERT INTO assessment_attempts (assessment_id, invite_id, user_id, status, time_limit_minutes)
		VALUES ($1, NULL, $2, 'pending', $3)
		RETURNING `+attemptColumns,
		assessmentID, userID, timeLimitMinutes)
	if err != nil {
		return models.Attempt{}, err
	}
	return attempt, nil
}

// GetAttempt loads one attempt by id.
func GetAttempt(ctx context.Context, db *sqlx.DB, id int64) (models.Attempt, error) {
	var attempt models.Attempt
	err := db.GetContext(ctx, &attempt, "SELECT "+attemptColumns+" FROM assessment_attempts WHERE id = $1", id)
	if err != nil {
		return models.Attempt{}, mapNotFound(err)
	}
	return attempt, nil
}

// MarkAttemptRunning flips a pending attempt to running with the given
// start time. A no-op (0 rows) if the attempt is no longer pending.
func MarkAttemptRunning(ctx context.Context, db *sqlx.DB, id int64, startedAt time.Time) error {
	_, err := db.ExecContext(ctx, `
		UPDATE assessment_attempts SET status = 'running', started_at = $2
		WHERE id = $1 AND status = 'pending'`, id, startedAt)
	return err
}

// MarkAttemptEnded flips an attempt to ended, whatever state it was in
// before (pending or running) — leaving a short history row rather than
// deleting it. A no-op (0 rows) if already ended.
func MarkAttemptEnded(ctx context.Context, db *sqlx.DB, id int64, endedAt time.Time) error {
	_, err := db.ExecContext(ctx, `
		UPDATE assessment_attempts SET status = 'ended', ended_at = $2
		WHERE id = $1 AND status != 'ended'`, id, endedAt)
	return err
}

// SetAttemptChapter updates the chapter cursor persisted alongside an
// attempt, so a reconnect (or a server restart) can resume from it.
func SetAttemptChapter(ctx context.Context, db *sqlx.DB, id, chapterID int64) error {
	_, err := db.ExecContext(ctx, "UPDATE assessment_attempts SET current_chapter_id = $2 WHERE id = $1", id, chapterID)
	return err
}

// LeaveAttempt marks an attempt ended in place (see MarkAttemptEnded) and
// returns the refreshed row; used when no live Runner exists to route a
// leaveCmd to.
func LeaveAttempt(ctx context.Context, db *sqlx.DB, id int64) (models.Attempt, error) {
	if err := MarkAttemptEnded(ctx, db, id, time.Now()); err != nil {
		return models.Attempt{}, err
	}
	return GetAttempt(ctx, db, id)
}

// UpsertChapterProgressStart records (or re-records) when the candidate
// entered a chapter.
func UpsertChapterProgressStart(ctx context.Context, db *sqlx.DB, attemptID, chapterID int64, startedAt time.Time) error {
	_, err := db.ExecContext(ctx, `
		INSERT INTO chapter_progress (attempt_id, chapter_id, started_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (attempt_id, chapter_id) DO UPDATE SET started_at = EXCLUDED.started_at`,
		attemptID, chapterID, startedAt)
	return err
}

// CompleteChapterProgress records when the candidate finished a chapter.
func CompleteChapterProgress(ctx context.Context, db *sqlx.DB, attemptID, chapterID int64, completedAt time.Time) error {
	_, err := db.ExecContext(ctx,
		"UPDATE chapter_progress SET completed_at = $3 WHERE attempt_id = $1 AND chapter_id = $2",
		attemptID, chapterID, completedAt)
	return err
}
