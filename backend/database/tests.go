package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

// testRow is one full row of the tests table (the hidden testing file lives
// here, next to its public name and description).
type testRow struct {
	ID           int64     `db:"id"`
	AssessmentID int64     `db:"assessment_id"`
	Name         string    `db:"name"`
	Description  string    `db:"description"`
	FileName     string    `db:"file_name"`
	FileContent  string    `db:"file_content"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

func (r testRow) test() models.Test {
	return models.Test{
		TestSummary: models.TestSummary{
			ID:          r.ID,
			Name:        r.Name,
			Description: r.Description,
		},
		AssessmentID: r.AssessmentID,
		File: models.CodeFile{
			FileName: r.FileName,
			Content:  r.FileContent,
		},
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// testColumns is the SELECT list for the full row shape above.
const testColumns = `id, assessment_id, name, description, file_name,
	file_content, created_at, updated_at`

// GetTest loads one hidden test including its testing file.
func GetTest(ctx context.Context, db *sqlx.DB, id int64) (models.Test, error) {
	var row testRow
	err := db.GetContext(ctx, &row, "SELECT "+testColumns+" FROM tests WHERE id = $1", id)
	if err != nil {
		return models.Test{}, mapNotFound(err)
	}
	return row.test(), nil
}

// AddTest adds a hidden test with its testing file to an assessment.
func AddTest(ctx context.Context, db *sqlx.DB, assessmentID int64, draft models.TestDraft) (models.Test, error) {
	var id int64
	err := db.GetContext(ctx, &id, `
		INSERT INTO tests (assessment_id, name, description, file_name, file_content)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		assessmentID, draft.Name, draft.Description, draft.File.FileName, draft.File.Content)
	if err != nil {
		return models.Test{}, err
	}
	return GetTest(ctx, db, id)
}

// TestPatch carries the optional updates of PATCH .../tests/{id}.
type TestPatch struct {
	Name        *string
	Description *string
	File        *models.CodeFile
}

// UpdateTest applies the provided fields and returns the refreshed row.
func UpdateTest(ctx context.Context, db *sqlx.DB, id int64, patch TestPatch) (models.Test, error) {
	sets := []string{}
	args := []any{}
	n := 0
	add := func(column string, value any) {
		n++
		sets = append(sets, fmt.Sprintf("%s = $%d", column, n))
		args = append(args, value)
	}
	if patch.Name != nil {
		add("name", *patch.Name)
	}
	if patch.Description != nil {
		add("description", *patch.Description)
	}
	if patch.File != nil {
		add("file_name", patch.File.FileName)
		add("file_content", patch.File.Content)
	}

	if len(sets) > 0 {
		n++
		query := "UPDATE tests SET " + strings.Join(sets, ", ") +
			fmt.Sprintf(" WHERE id = $%d", n)
		res, err := db.ExecContext(ctx, query, append(args, id)...)
		if err != nil {
			return models.Test{}, err
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return models.Test{}, ErrNotFound
		}
	}
	return GetTest(ctx, db, id)
}

// DeleteTest removes one hidden test.
func DeleteTest(ctx context.Context, db *sqlx.DB, id int64) error {
	res, err := db.ExecContext(ctx, "DELETE FROM tests WHERE id = $1", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
