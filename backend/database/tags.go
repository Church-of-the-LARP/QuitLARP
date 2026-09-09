package database

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"

	"backend/models"
)

// ensureTag resolves a tag by case-insensitive name, creating it when
// missing. Concurrent creates are handled by re-running the lookup after a
// unique violation (SQLSTATE 23505).
func ensureTag(ctx context.Context, q dbtx, name string) (int64, error) {
	name = strings.TrimSpace(name)
	var id int64
	err := q.GetContext(ctx, &id, "SELECT id FROM tags WHERE LOWER(name) = LOWER($1)", name)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}
	err = q.GetContext(ctx, &id, "INSERT INTO tags (name) VALUES ($1) RETURNING id", name)
	if err == nil {
		return id, nil
	}
	if !IsUniqueViolation(err) {
		return 0, err
	}
	// Lost the race against a concurrent create; fetch the winner instead.
	err = q.GetContext(ctx, &id, "SELECT id FROM tags WHERE LOWER(name) = LOWER($1)", name)
	return id, err
}

// ReplaceAssessmentTags sets the full tag list of an assessment in one
// transaction. Unknown tag names are created on the fly; blank or duplicate
// names in the input are ignored.
func ReplaceAssessmentTags(ctx context.Context, db *sqlx.DB, assessmentID int64, names []string) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx,
		"DELETE FROM assessment_tags WHERE assessment_id = $1", assessmentID); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, raw := range names {
		name := strings.TrimSpace(raw)
		if name == "" || seen[strings.ToLower(name)] {
			continue
		}
		seen[strings.ToLower(name)] = true
		if err := linkTag(ctx, tx, assessmentID, name); err != nil {
			return err
		}
	}
	return tx.Commit()
}

// ListTags returns every tag with its assessment count, ordered by name.
func ListTags(ctx context.Context, db *sqlx.DB) ([]models.TagCount, error) {
	tags := []models.TagCount{}
	err := db.SelectContext(ctx, &tags, `
		SELECT t.id, t.name, COUNT(at.assessment_id) AS assessment_count
		FROM tags t
		LEFT JOIN assessment_tags at ON at.tag_id = t.id
		GROUP BY t.id, t.name
		ORDER BY LOWER(t.name)`)
	return tags, err
}
