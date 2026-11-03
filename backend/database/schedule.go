package database

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

// ListUpcomingAssessments returns the user's scheduled assessments that have
// not started yet, soonest first.
func ListUpcomingAssessments(ctx context.Context, db *sqlx.DB, userID int64, limit int) ([]models.ScheduledAssessment, error) {
	out := []models.ScheduledAssessment{}
	err := db.SelectContext(ctx, &out, `
		SELECT s.id, s.assessment_id, a.title, a.difficulty, a.time_limit_minutes,
		       (SELECT COUNT(*) FROM tests t WHERE t.assessment_id = a.id) AS test_count,
		       s.scheduled_at
		FROM scheduled_assessments s
		JOIN assessments a ON a.id = s.assessment_id
		WHERE s.user_id = $1 AND s.scheduled_at >= NOW()
		ORDER BY s.scheduled_at ASC, s.id ASC
		LIMIT $2`, userID, limit)
	return out, err
}

// SaveScheduledAssessment plans an assessment for the user, moving the time
// when an entry already exists. It returns ErrNotFound for an unknown
// assessment.
func SaveScheduledAssessment(ctx context.Context, db *sqlx.DB, userID, assessmentID int64, at time.Time) (models.ScheduledAssessment, error) {
	var id int64
	err := db.GetContext(ctx, &id, `
		INSERT INTO scheduled_assessments (user_id, assessment_id, scheduled_at)
		SELECT $1, a.id, $3 FROM assessments a WHERE a.id = $2
		ON CONFLICT (user_id, assessment_id) DO UPDATE SET scheduled_at = EXCLUDED.scheduled_at
		RETURNING id`, userID, assessmentID, at)
	if err != nil {
		return models.ScheduledAssessment{}, mapNotFound(err)
	}
	var out models.ScheduledAssessment
	err = db.GetContext(ctx, &out, `
		SELECT s.id, s.assessment_id, a.title, a.difficulty, a.time_limit_minutes,
		       (SELECT COUNT(*) FROM tests t WHERE t.assessment_id = a.id) AS test_count,
		       s.scheduled_at
		FROM scheduled_assessments s
		JOIN assessments a ON a.id = s.assessment_id
		WHERE s.id = $1`, id)
	return out, err
}

// DeleteScheduledAssessment removes one of the user's entries. It returns
// ErrNotFound when the entry does not exist or belongs to someone else.
func DeleteScheduledAssessment(ctx context.Context, db *sqlx.DB, userID, id int64) error {
	res, err := db.ExecContext(ctx,
		"DELETE FROM scheduled_assessments WHERE id = $1 AND user_id = $2", id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
