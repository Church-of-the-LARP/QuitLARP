package database

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"backend/assessment"
	"backend/models"
)

// dbtx is satisfied by both *sqlx.DB and *sqlx.Tx so row-level helpers can
// run inside the transactions of their callers.
type dbtx interface {
	sqlx.ExtContext
	GetContext(ctx context.Context, dest any, query string, args ...any) error
	SelectContext(ctx context.Context, dest any, query string, args ...any) error
}

// assessmentRow is one row of assessments joined with its author's username
// (nil when the author account was deleted).
type assessmentRow struct {
	ID               int64             `db:"id"`
	Title            string            `db:"title"`
	Description      string            `db:"description"`
	Kind             string            `db:"kind"`
	Difficulty       models.Difficulty `db:"difficulty"`
	TimeLimitMinutes int               `db:"time_limit_minutes"`
	TemplateFileName string            `db:"template_file_name"`
	TemplateContent  string            `db:"template_file_content"`
	AuthorID         *int64            `db:"author_id"`
	AuthorUsername   *string           `db:"author_username"`
	CreatedAt        time.Time         `db:"created_at"`
	UpdatedAt        time.Time         `db:"updated_at"`
}

// assessmentColumns is the SELECT list for the joined row shape above.
const assessmentColumns = `
	a.id, a.title, a.description, a.kind, a.difficulty, a.time_limit_minutes,
	a.template_file_name, a.template_file_content, a.author_id,
	u.username AS author_username, a.created_at, a.updated_at`

func (r assessmentRow) author() *models.AuthorRef {
	if r.AuthorID == nil {
		return nil
	}
	return &models.AuthorRef{ID: *r.AuthorID, Username: *r.AuthorUsername}
}

func (r assessmentRow) summary() models.AssessmentSummary {
	return models.AssessmentSummary{
		ID:               r.ID,
		Title:            r.Title,
		Description:      r.Description,
		Kind:             r.Kind,
		Difficulty:       r.Difficulty,
		TimeLimitMinutes: r.TimeLimitMinutes,
		TemplateFileName: r.TemplateFileName,
		Author:           r.author(),
		Tags:             []models.Tag{},
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}
}

func (r assessmentRow) assessment() models.Assessment {
	return models.Assessment{
		ID:               r.ID,
		Title:            r.Title,
		Description:      r.Description,
		Kind:             r.Kind,
		Difficulty:       r.Difficulty,
		TimeLimitMinutes: r.TimeLimitMinutes,
		Template: models.CodeFile{
			FileName: r.TemplateFileName,
			Content:  r.TemplateContent,
		},
		Author:    r.author(),
		Tags:      []models.Tag{},
		Chapters:  []models.Chapter{},
		Tests:     []models.TestSummary{},
		CreatedAt: r.CreatedAt,
		UpdatedAt: r.UpdatedAt,
	}
}

// AssessmentFilter narrows GET /assessments. Nil Difficulty and empty
// Tag/Q mean "no filter on that axis".
type AssessmentFilter struct {
	Difficulty *models.Difficulty
	Tag        string
	Q          string
	Offset     int
	Limit      int
}

// assessmentWhere renders the shared WHERE clause for list and count
// queries. Conditions use the table alias "a".
func assessmentWhere(f AssessmentFilter) (string, []any) {
	conds := []string{}
	args := []any{}
	if f.Difficulty != nil {
		args = append(args, string(*f.Difficulty))
		conds = append(conds, fmt.Sprintf("a.difficulty = $%d", len(args)))
	}
	if f.Tag != "" {
		args = append(args, f.Tag)
		conds = append(conds, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM assessment_tags at JOIN tags t ON t.id = at.tag_id "+
				"WHERE at.assessment_id = a.id AND LOWER(t.name) = LOWER($%d))", len(args)))
	}
	if f.Q != "" {
		args = append(args, "%"+f.Q+"%")
		conds = append(conds, fmt.Sprintf(
			"(a.title ILIKE $%d OR a.description ILIKE $%d)", len(args), len(args)))
	}
	if len(conds) == 0 {
		return "", nil
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// ListAssessments returns assessment summaries ordered newest first. The
// caller fills no nested content; tags are attached per row.
func ListAssessments(ctx context.Context, db *sqlx.DB, f AssessmentFilter) ([]models.AssessmentSummary, error) {
	where, args := assessmentWhere(f)
	query := "SELECT" + assessmentColumns +
		" FROM assessments a LEFT JOIN users u ON u.id = a.author_id" + where +
		fmt.Sprintf(" ORDER BY a.id DESC LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
	args = append(args, f.Limit, f.Offset)

	rows := []assessmentRow{}
	if err := db.SelectContext(ctx, &rows, query, args...); err != nil {
		return nil, err
	}

	summaries := make([]models.AssessmentSummary, 0, len(rows))
	ids := make([]int64, 0, len(rows))
	for _, r := range rows {
		summaries = append(summaries, r.summary())
		ids = append(ids, r.ID)
	}
	tagMap, err := listAssessmentTags(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	for i := range summaries {
		tags := tagMap[summaries[i].ID]
		if tags == nil {
			tags = []models.Tag{}
		}
		summaries[i].Tags = tags
	}
	return summaries, nil
}

// CountAssessments returns the number of assessments matching the filter
// (used for pagination metadata alongside ListAssessments).
func CountAssessments(ctx context.Context, db *sqlx.DB, f AssessmentFilter) (int64, error) {
	where, args := assessmentWhere(f)
	query := "SELECT COUNT(*) FROM assessments a" + where
	var total int64
	if err := db.GetContext(ctx, &total, query, args...); err != nil {
		return 0, err
	}
	return total, nil
}

// listAssessmentTags fetches the tags of the given assessments in one query,
// grouped by assessment id. Empty ids yield an empty map.
func listAssessmentTags(ctx context.Context, q dbtx, ids []int64) (map[int64][]models.Tag, error) {
	out := map[int64][]models.Tag{}
	if len(ids) == 0 {
		return out, nil
	}
	rows := []struct {
		AssessmentID int64  `db:"assessment_id"`
		TagID        int64  `db:"tag_id"`
		Name         string `db:"name"`
	}{}
	err := q.SelectContext(ctx, &rows, `
		SELECT at.assessment_id, t.id AS tag_id, t.name
		FROM assessment_tags at JOIN tags t ON t.id = at.tag_id
		WHERE at.assessment_id = ANY($1)
		ORDER BY t.name`, ids)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.AssessmentID] = append(out[r.AssessmentID], models.Tag{ID: r.TagID, Name: r.Name})
	}
	return out, nil
}

// GetAssessmentDetail loads an assessment with its tags, chapters and test
// summaries (tests never include the hidden file content here).
func GetAssessmentDetail(ctx context.Context, db *sqlx.DB, id int64) (models.Assessment, error) {
	query := "SELECT" + assessmentColumns +
		" FROM assessments a LEFT JOIN users u ON u.id = a.author_id WHERE a.id = $1"
	var row assessmentRow
	if err := db.GetContext(ctx, &row, query, id); err != nil {
		return models.Assessment{}, mapNotFound(err)
	}
	got := row.assessment()

	tagMap, err := listAssessmentTags(ctx, db, []int64{id})
	if err != nil {
		return models.Assessment{}, err
	}
	got.Tags = tagMap[id]
	if got.Tags == nil {
		got.Tags = []models.Tag{}
	}

	if err := db.SelectContext(ctx, &got.Chapters, `
		SELECT id, assessment_id, position, title, description,
		       time_limit_minutes, start_mode, task_file, created_at, updated_at
		FROM chapters WHERE assessment_id = $1 ORDER BY position, id`, id); err != nil {
		return models.Assessment{}, err
	}
	if err := db.SelectContext(ctx, &got.Tests, `
		SELECT id, name, description
		FROM tests WHERE assessment_id = $1 ORDER BY id`, id); err != nil {
		return models.Assessment{}, err
	}
	return got, nil
}

// GetAssessmentAuthorID returns the author id of an assessment (nil once the
// author account has been deleted), or ErrNotFound.
func GetAssessmentAuthorID(ctx context.Context, db *sqlx.DB, id int64) (*int64, error) {
	var authorID *int64
	err := db.GetContext(ctx, &authorID, "SELECT author_id FROM assessments WHERE id = $1", id)
	if err != nil {
		return nil, mapNotFound(err)
	}
	return authorID, nil
}

// CreateAssessment inserts an assessment together with its chapters, hidden
// tests and tags in one transaction. Unknown tag names are created on the
// fly. The draft is expected to be validated by the caller.
func CreateAssessment(ctx context.Context, db *sqlx.DB, authorID int64, draft models.AssessmentDraft) (models.Assessment, error) {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return models.Assessment{}, err
	}
	defer tx.Rollback()

	var id int64
	err = tx.GetContext(ctx, &id, `
		INSERT INTO assessments (title, description, difficulty, time_limit_minutes,
		                         template_file_name, template_file_content, author_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		draft.Title, draft.Description, draft.Difficulty, draft.TimeLimitMinutes,
		draft.Template.FileName, draft.Template.Content, authorID)
	if err != nil {
		return models.Assessment{}, err
	}

	for i, ch := range draft.Chapters {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO chapters (assessment_id, position, title, description, time_limit_minutes)
			VALUES ($1, $2, $3, $4, $5)`,
			id, i+1, ch.Title, ch.Description, ch.TimeLimitMinutes); err != nil {
			return models.Assessment{}, err
		}
	}
	for _, t := range draft.Tests {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO tests (assessment_id, name, description, file_name, file_content)
			VALUES ($1, $2, $3, $4, $5)`,
			id, t.Name, t.Description, t.File.FileName, t.File.Content); err != nil {
			return models.Assessment{}, err
		}
	}
	for _, name := range draft.Tags {
		if err := linkTag(ctx, tx, id, name); err != nil {
			return models.Assessment{}, err
		}
	}

	if err := tx.Commit(); err != nil {
		return models.Assessment{}, err
	}
	return GetAssessmentDetail(ctx, db, id)
}

// linkTag attaches one tag (by name, auto-creating it when missing) to an
// assessment; duplicates are ignored.
func linkTag(ctx context.Context, q dbtx, assessmentID int64, name string) error {
	tagID, err := ensureTag(ctx, q, name)
	if err != nil {
		return err
	}
	_, err = q.ExecContext(ctx, `
		INSERT INTO assessment_tags (assessment_id, tag_id)
		VALUES ($1, $2) ON CONFLICT DO NOTHING`, assessmentID, tagID)
	return err
}

// AssessmentPatch carries the optional scalar updates of PATCH /assessments.
type AssessmentPatch struct {
	Title            *string
	Description      *string
	Difficulty       *models.Difficulty
	TimeLimitMinutes *int
	Template         *models.CodeFile
}

// UpdateAssessment applies the provided fields and returns the refreshed
// detail. A patch with no fields set is still validated for existence.
func UpdateAssessment(ctx context.Context, db *sqlx.DB, id int64, patch AssessmentPatch) (models.Assessment, error) {
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
	if patch.Difficulty != nil {
		add("difficulty", string(*patch.Difficulty))
	}
	if patch.TimeLimitMinutes != nil {
		add("time_limit_minutes", *patch.TimeLimitMinutes)
	}
	if patch.Template != nil {
		add("template_file_name", patch.Template.FileName)
		add("template_file_content", patch.Template.Content)
	}

	if len(sets) > 0 {
		n++
		query := "UPDATE assessments SET " + strings.Join(sets, ", ") +
			fmt.Sprintf(" WHERE id = $%d", n)
		res, err := db.ExecContext(ctx, query, append(args, id)...)
		if err != nil {
			return models.Assessment{}, err
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return models.Assessment{}, ErrNotFound
		}
	}
	return GetAssessmentDetail(ctx, db, id)
}

// DeleteAssessment removes an assessment; its chapters, tests and tag links
// go with it via cascade.
func DeleteAssessment(ctx context.Context, db *sqlx.DB, id int64) error {
	res, err := db.ExecContext(ctx, "DELETE FROM assessments WHERE id = $1", id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// defaultChapterTimeLimitMinutes is the limit new repository-synced chapters
// get: the repository format carries no time limit yet, so new chapters take
// the platform default while existing values stay untouched.
const defaultChapterTimeLimitMinutes = 30

// GetAssessmentMainCommit returns the main commit sha the assessment was last
// synced from; "" when it was never synced. ErrNotFound when the assessment
// does not exist.
func GetAssessmentMainCommit(ctx context.Context, db *sqlx.DB, id int64) (string, error) {
	var commit *string
	err := db.GetContext(ctx, &commit, "SELECT main_commit FROM assessments WHERE id = $1", id)
	if err != nil {
		return "", mapNotFound(err)
	}
	if commit == nil {
		return "", nil
	}
	return *commit, nil
}

// SyncAssessmentContent replaces the repository-derived content of an
// assessment: kind, description, chapters (matched by position) and the sync
// bookkeeping. Manually managed fields (title, difficulty, time limit, tags,
// tests, template) are untouched. Runs in one transaction.
func SyncAssessmentContent(ctx context.Context, db *sqlx.DB, id int64, content assessment.Content, mainCommit string) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin sync of assessment %d: %w", id, err)
	}
	defer tx.Rollback()

	res, err := tx.ExecContext(ctx, `
		UPDATE assessments
		SET description = $2, kind = $3, main_commit = $4, synced_at = NOW()
		WHERE id = $1`, id, content.Description, content.Kind, mainCommit)
	if err != nil {
		return fmt.Errorf("sync assessment %d: %w", id, err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return ErrNotFound
	}

	existing, err := chapterIDsByPosition(ctx, tx, id)
	if err != nil {
		return fmt.Errorf("load chapters of assessment %d: %w", id, err)
	}

	chapters := make([]assessment.Chapter, len(content.Chapters))
	copy(chapters, content.Chapters)
	sort.SliceStable(chapters, func(i, j int) bool { return chapters[i].Index < chapters[j].Index })

	kept := map[int]bool{}
	for _, ch := range chapters {
		kept[ch.Index] = true
		if _, ok := existing[ch.Index]; ok {
			if _, err := tx.ExecContext(ctx, `
				UPDATE chapters
				SET title = $2, description = $3, start_mode = $4, task_file = $5
				WHERE assessment_id = $1 AND position = $6`,
				id, ch.Name, ch.Readme, ch.Start, ch.Task, ch.Index); err != nil {
				return fmt.Errorf("update chapter at position %d of assessment %d: %w", ch.Index, id, err)
			}
			continue
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO chapters (assessment_id, position, title, description,
			                      start_mode, task_file, time_limit_minutes)
			VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			id, ch.Index, ch.Name, ch.Readme, ch.Start, ch.Task, defaultChapterTimeLimitMinutes); err != nil {
			return fmt.Errorf("insert chapter at position %d of assessment %d: %w", ch.Index, id, err)
		}
	}

	stale := []int64{}
	for position, chapterID := range existing {
		if !kept[position] {
			stale = append(stale, chapterID)
		}
	}
	if len(stale) > 0 {
		if _, err := tx.ExecContext(ctx,
			"DELETE FROM chapters WHERE assessment_id = $1 AND id = ANY($2)", id, stale); err != nil {
			return fmt.Errorf("delete stale chapters of assessment %d: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sync of assessment %d: %w", id, err)
	}
	return nil
}

// chapterIDsByPosition maps the position of each existing chapter of an
// assessment to its id.
func chapterIDsByPosition(ctx context.Context, q dbtx, assessmentID int64) (map[int]int64, error) {
	rows := []struct {
		ID       int64 `db:"id"`
		Position int   `db:"position"`
	}{}
	if err := q.SelectContext(ctx, &rows,
		"SELECT id, position FROM chapters WHERE assessment_id = $1", assessmentID); err != nil {
		return nil, err
	}
	out := make(map[int]int64, len(rows))
	for _, r := range rows {
		out[r.Position] = r.ID
	}
	return out, nil
}
