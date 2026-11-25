package database

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jmoiron/sqlx"

	"backend/assessment"
	"backend/models"
	"backend/repos"
)

// seedData holds the assessment trees that SeedAssessments publishes on
// first boot. all: is required because the TwoSum tree carries a .gitignore.
//
//go:embed all:seeddata
var seedData embed.FS

// seedAssessment is one built-in assessment. Key is the stable identity that
// survives restarts; the files live in seeddata/<Key>/.
type seedAssessment struct {
	Key              string
	Title            string
	Difficulty       models.Difficulty
	TimeLimitMinutes int
	Tags             []string
	TemplateFile     string
}

var seededAssessments = []seedAssessment{
	{
		Key:              "twosum",
		Title:            "Two Sum",
		Difficulty:       models.DifficultyEasy,
		TimeLimitMinutes: 120,
		Tags:             []string{"arrays", "hash-map", "python"},
		TemplateFile:     "chapters/1_two-sum/glue/two_sum_functions.py",
	},
	{
		Key:              "petrik-verseny",
		Title:            "Petrik Verseny 2026 – Kisváros weboldal",
		Difficulty:       models.DifficultyEasy,
		TimeLimitMinutes: 120,
		Tags:             []string{"petrik", "verseny", "html", "css", "bootstrap"},
		TemplateFile:     "deploy/index.html",
	},
}

// SeedAssessments guarantees the built-in assessments exist with their git
// repositories on main. It is idempotent: an existing seed row is reused, and
// an existing repository is never rewritten, so a later push to main wins over
// the embedded starter tree.
func SeedAssessments(ctx context.Context, db *sqlx.DB, reposDir string, authorID int64) error {
	for _, seed := range seededAssessments {
		if err := seedOneAssessment(ctx, db, reposDir, authorID, seed); err != nil {
			return fmt.Errorf("seed assessment %q: %w", seed.Key, err)
		}
	}
	return nil
}

func seedOneAssessment(ctx context.Context, db *sqlx.DB, reposDir string, authorID int64, seed seedAssessment) error {
	files, err := seedFiles(seed.Key)
	if err != nil {
		return err
	}
	content, err := seedContent(seed.Key, files)
	if err != nil {
		return err
	}
	if strings.TrimSpace(content.Description) == "" {
		return errors.New("README.md must describe the assessment")
	}

	templateName := seed.TemplateFile
	if templateName == "" {
		templateName = "main.py"
	}
	template, ok := files[templateName]
	if !ok {
		return fmt.Errorf("template file %q is missing from seeddata/%s", templateName, seed.Key)
	}

	id, created, err := ensureSeededAssessment(ctx, db, authorID, seed, content, templateName, string(template))
	if err != nil {
		return err
	}

	repoDir := filepath.Join(reposDir, filepath.FromSlash(models.AssessmentRepoID(id)+".git"))
	sha, err := mainRevision(repoDir)
	if err != nil {
		return fmt.Errorf("read repository: %w", err)
	}
	if sha == "" {
		sha, err = repos.CommitFiles(repoDir, files,
			"Initial content for "+seed.Title, seedSignature())
		if err != nil {
			return fmt.Errorf("write repository: %w", err)
		}
	}

	current, err := GetAssessmentMainCommit(ctx, db, id)
	if err != nil {
		return err
	}
	if created || current != sha {
		if err := SyncAssessmentContent(ctx, db, id, content, sha); err != nil {
			return fmt.Errorf("sync content: %w", err)
		}
	}
	return nil
}

// ensureSeededAssessment returns the assessment carrying seed.Key, creating it
// (and its tags) when missing. The second result reports whether this call
// created the row.
func ensureSeededAssessment(ctx context.Context, db *sqlx.DB, authorID int64, seed seedAssessment, content assessment.Content, templateName, templateContent string) (int64, bool, error) {
	var id int64
	err := db.GetContext(ctx, &id, "SELECT id FROM assessments WHERE seed_key = $1", seed.Key)
	switch {
	case err == nil:
		return id, false, nil
	case !errors.Is(err, sql.ErrNoRows):
		return 0, false, err
	}

	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()

	err = tx.GetContext(ctx, &id, `
		INSERT INTO assessments (title, description, difficulty, time_limit_minutes,
		                         template_file_name, template_file_content, author_id,
		                         kind, seed_key)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		seed.Title, content.Description, seed.Difficulty, seed.TimeLimitMinutes,
		templateName, templateContent, authorID, content.Kind, seed.Key)
	if err != nil {
		if IsUniqueViolation(err) {
			tx.Rollback()
			if selectErr := db.GetContext(ctx, &id,
				"SELECT id FROM assessments WHERE seed_key = $1", seed.Key); selectErr == nil {
				return id, false, nil
			}
			return 0, false, err
		}
		return 0, false, err
	}
	for _, name := range seed.Tags {
		if err := linkTag(ctx, tx, id, name); err != nil {
			return 0, false, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, false, err
	}
	return id, true, nil
}

// seedFiles reads one embedded assessment tree, keyed by slash path relative
// to its root.
func seedFiles(key string) (map[string][]byte, error) {
	root := path.Join("seeddata", key)
	files := map[string][]byte{}
	err := fs.WalkDir(seedData, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := seedData.ReadFile(p)
		if err != nil {
			return err
		}
		files[strings.TrimPrefix(p, root+"/")] = data
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("read embedded seeddata/%s: %w", key, err)
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("embedded seeddata/%s is empty", key)
	}
	return files, nil
}

// seedContent materializes files in a temp directory and runs the same layout
// validation a push does, so a broken seed tree fails at boot instead of
// producing an assessment the rest of the platform cannot use.
func seedContent(key string, files map[string][]byte) (assessment.Content, error) {
	tmp, err := os.MkdirTemp("", "assessment-seed-"+key+"-*")
	if err != nil {
		return assessment.Content{}, err
	}
	defer os.RemoveAll(tmp)

	for name, data := range files {
		dest := filepath.Join(tmp, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return assessment.Content{}, err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return assessment.Content{}, err
		}
	}
	content, violations := assessment.Load(tmp)
	if len(violations) > 0 {
		return assessment.Content{}, fmt.Errorf("invalid layout: %s", strings.Join(violations, "; "))
	}
	return content, nil
}

// mainRevision returns the main commit of the bare repository at dir, or ""
// when the repository (or its main branch) does not exist yet.
func mainRevision(dir string) (string, error) {
	if _, err := os.Stat(filepath.Join(dir, "HEAD")); err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	repo, err := repos.OpenDir(dir)
	if err != nil {
		return "", err
	}
	sha, err := repo.Revision("main")
	if errors.Is(err, repos.ErrRefNotFound) {
		return "", nil
	}
	return sha, err
}

func seedSignature() object.Signature {
	return object.Signature{Name: "QuitLARP", Email: "seed@quitlarp.local", When: time.Now().UTC()}
}
