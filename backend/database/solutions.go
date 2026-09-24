package database

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

// solutionRow is one row of solutions joined with the solver's username.
type solutionRow struct {
	ID           int64     `db:"id"`
	AssessmentID int64     `db:"assessment_id"`
	UserID       int64     `db:"user_id"`
	Public       bool      `db:"is_public"`
	CommitSHA    string    `db:"commit_sha"`
	Username     *string   `db:"username"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// solutionColumns is the SELECT list matching solutionRow.
const solutionColumns = `
	s.id, s.assessment_id, s.user_id, s.is_public, s.commit_sha,
	u.username, s.created_at, s.updated_at`

func (r solutionRow) solution() models.Solution {
	var author *models.AuthorRef
	if r.Username != nil {
		author = &models.AuthorRef{ID: r.UserID, Username: *r.Username}
	}
	return models.Solution{
		ID:           r.ID,
		AssessmentID: r.AssessmentID,
		UserID:       r.UserID,
		Public:       r.Public,
		CommitSHA:    r.CommitSHA,
		Author:       author,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}
}

// SaveSolution records one submission, replacing the previous commit and
// visibility when the user already has a solution for the assessment.
func SaveSolution(ctx context.Context, db *sqlx.DB, assessmentID, userID int64, commitSHA string, public bool) (models.Solution, error) {
	_, err := db.ExecContext(ctx, `
		INSERT INTO solutions (assessment_id, user_id, commit_sha, is_public)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (assessment_id, user_id) DO UPDATE
		SET commit_sha = EXCLUDED.commit_sha, is_public = EXCLUDED.is_public`,
		assessmentID, userID, commitSHA, public)
	if err != nil {
		return models.Solution{}, err
	}
	return GetSolution(ctx, db, assessmentID, userID)
}

// GetSolution returns one user's solution for an assessment.
func GetSolution(ctx context.Context, db *sqlx.DB, assessmentID, userID int64) (models.Solution, error) {
	return querySolution(ctx, db, "s.assessment_id = $1 AND s.user_id = $2", assessmentID, userID)
}

// GetSolutionByID returns a solution by its own id.
func GetSolutionByID(ctx context.Context, db *sqlx.DB, id int64) (models.Solution, error) {
	return querySolution(ctx, db, "s.id = $1", id)
}

// SetSolutionPublic changes whether a solution is readable by every
// signed-in user.
func SetSolutionPublic(ctx context.Context, db *sqlx.DB, id int64, public bool) (models.Solution, error) {
	res, err := db.ExecContext(ctx, "UPDATE solutions SET is_public = $2 WHERE id = $1", id, public)
	if err != nil {
		return models.Solution{}, err
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return models.Solution{}, ErrNotFound
	}
	return GetSolutionByID(ctx, db, id)
}

// ListSolutions returns every solution of an assessment, newest first.
func ListSolutions(ctx context.Context, db *sqlx.DB, assessmentID int64) ([]models.Solution, error) {
	rows := []solutionRow{}
	err := db.SelectContext(ctx, &rows,
		"SELECT"+solutionColumns+` FROM solutions s
		 LEFT JOIN users u ON u.id = s.user_id
		 WHERE s.assessment_id = $1
		 ORDER BY s.updated_at DESC`, assessmentID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Solution, 0, len(rows))
	for _, r := range rows {
		out = append(out, r.solution())
	}
	return out, nil
}

// querySolution runs a solution lookup with a fixed WHERE clause built by
// this package only.
func querySolution(ctx context.Context, db *sqlx.DB, where string, args ...any) (models.Solution, error) {
	var row solutionRow
	err := db.GetContext(ctx, &row,
		"SELECT"+solutionColumns+` FROM solutions s
		 LEFT JOIN users u ON u.id = s.user_id
		 WHERE `+where, args...)
	if err != nil {
		return models.Solution{}, mapNotFound(err)
	}
	return row.solution(), nil
}
