package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

// chapterColumns is the SELECT list for a full chapter row.
const chapterColumns = `id, assessment_id, position, title, description,
	time_limit_minutes, start_mode, created_at, updated_at`

// GetChapter loads one chapter by id.
func GetChapter(ctx context.Context, db *sqlx.DB, id int64) (models.Chapter, error) {
	var ch models.Chapter
	err := db.GetContext(ctx, &ch, "SELECT "+chapterColumns+" FROM chapters WHERE id = $1", id)
	if err != nil {
		return models.Chapter{}, mapNotFound(err)
	}
	return ch, nil
}

// AddChapter appends a chapter to an assessment at the next free position
// (the position of the last chapter plus one).
func AddChapter(ctx context.Context, db *sqlx.DB, assessmentID int64, draft models.ChapterDraft) (models.Chapter, error) {
	var id int64
	err := db.GetContext(ctx, &id, `
		INSERT INTO chapters (assessment_id, position, title, description, time_limit_minutes)
		SELECT $1, COALESCE(MAX(position), 0) + 1, $2, $3, $4
		FROM chapters WHERE assessment_id = $1
		RETURNING id`,
		assessmentID, draft.Title, draft.Description, draft.TimeLimitMinutes)
	if err != nil {
		return models.Chapter{}, err
	}
	return GetChapter(ctx, db, id)
}

// ChapterPatch carries the optional updates of PATCH .../chapters/{id}.
type ChapterPatch struct {
	Title            *string
	Description      *string
	TimeLimitMinutes *int
	Position         *int
}

// UpdateChapter applies the provided fields and returns the refreshed row.
func UpdateChapter(ctx context.Context, db *sqlx.DB, id int64, patch ChapterPatch) (models.Chapter, error) {
	sets := []string{}
	args := []any{}
	n := 0
	add := func(column string, value any) {
		n++
		sets = append(sets, fmt.Sprintf("%s = $%d", column, n))
		args = append(args, value)
	}
	if patch.Title != nil {
		add("title", *patch.Title)
	}
	if patch.Description != nil {
		add("description", *patch.Description)
	}
	if patch.TimeLimitMinutes != nil {
		add("time_limit_minutes", *patch.TimeLimitMinutes)
	}
	if patch.Position != nil {
		add("position", *patch.Position)
	}

	if len(sets) > 0 {
		n++
		query := "UPDATE chapters SET " + strings.Join(sets, ", ") +
			fmt.Sprintf(" WHERE id = $%d", n)
		res, err := db.ExecContext(ctx, query, append(args, id)...)
		if err != nil {
			return models.Chapter{}, err
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return models.Chapter{}, ErrNotFound
		}
	}
	return GetChapter(ctx, db, id)
}

// DeleteChapter removes one chapter.
func DeleteChapter(ctx context.Context, db *sqlx.DB, id int64) error {
	res, err := db.ExecContext(ctx, "DELETE FROM chapters WHERE id = $1", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}
