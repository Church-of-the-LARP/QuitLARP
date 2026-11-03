package database

import (
	"context"
	"time"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

type AdminSolution struct {
	ID             int64             `db:"id"`
	AssessmentID   int64             `db:"assessment_id"`
	AssessmentTitle string            `db:"assessment_title"`
	UserID         int64             `db:"user_id"`
	Username       *string           `db:"username"`
	Public         bool              `db:"is_public"`
	CommitSHA      string            `db:"commit_sha"`
	CreatedAt      time.Time         `db:"created_at"`
	UpdatedAt      time.Time         `db:"updated_at"`
	Author         *models.AuthorRef `db:"-"`
}

func ListAllSolutions(ctx context.Context, db *sqlx.DB) ([]AdminSolution, error) {
	rows := []AdminSolution{}
	err := db.SelectContext(ctx, &rows, `
		SELECT s.id, s.assessment_id, a.title AS assessment_title,
		       s.user_id, u.username, s.is_public, s.commit_sha,
		       s.created_at, s.updated_at
		FROM solutions s
		LEFT JOIN assessments a ON a.id = s.assessment_id
		LEFT JOIN users u ON u.id = s.user_id
		ORDER BY s.updated_at DESC`)
	if err != nil {
		return nil, err
	}
	for i := range rows {
		if rows[i].Username != nil {
			rows[i].Author = &models.AuthorRef{ID: rows[i].UserID, Username: *rows[i].Username}
		}
	}
	return rows, nil
}
