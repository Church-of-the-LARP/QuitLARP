package database

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

// SaveTestRun stores one run-tests outcome for the user and assessment. The
// caller has already authorized the session.
func SaveTestRun(ctx context.Context, db *sqlx.DB, userID, assessmentID int64, results any, allPassed bool) error {
	encoded, err := json.Marshal(results)
	if err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
		INSERT INTO assessment_test_runs (user_id, assessment_id, results, all_passed)
		VALUES ($1, $2, $3, $4)`, userID, assessmentID, encoded, allPassed)
	return err
}

// LatestTestRun returns the most recent stored run for a user and assessment.
// ErrNotFound when they have not run the checks yet.
func LatestTestRun(ctx context.Context, db *sqlx.DB, assessmentID, userID int64) (models.AssessmentTestRun, error) {
	var row struct {
		ID           int64           `db:"id"`
		AssessmentID int64           `db:"assessment_id"`
		Results      json.RawMessage `db:"results"`
		AllPassed    bool            `db:"all_passed"`
		CreatedAt    time.Time       `db:"created_at"`
	}
	err := db.GetContext(ctx, &row, `
		SELECT id, assessment_id, results, all_passed, created_at
		FROM assessment_test_runs
		WHERE assessment_id = $1 AND user_id = $2
		ORDER BY created_at DESC, id DESC
		LIMIT 1`, assessmentID, userID)
	if err != nil {
		return models.AssessmentTestRun{}, mapNotFound(err)
	}
	out := models.AssessmentTestRun{
		ID:           row.ID,
		AssessmentID: row.AssessmentID,
		AllPassed:    row.AllPassed,
		CreatedAt:    row.CreatedAt,
	}
	if len(row.Results) > 0 {
		if err := json.Unmarshal(row.Results, &out.Results); err != nil {
			return models.AssessmentTestRun{}, err
		}
	}
	return out, nil
}
