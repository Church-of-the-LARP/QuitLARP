package sessions

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jmoiron/sqlx"

	"backend/config"
	"backend/database"
	"backend/models"
	"backend/repos"
	"backend/supervisor"
)

const (
	// sessionLabelRole marks every container this package creates, so the
	// cleanup pass can find the leftovers of a previous backend process.
	sessionLabelRole = "codingtest.role"
	sessionRole      = "assessment-session"
	janitorInterval  = 30 * time.Second
)

// sessionKey identifies the one session a user may hold per assessment.
type sessionKey struct {
	userID       int64
	assessmentID int64
}

// Manager owns every live session of this process. Sessions are in-memory by
// design: a backend restart drops them and the next boot removes their
// containers. Submitted solutions live in the database and in git, so
// nothing of value is held here.
type Manager struct {
	cfg     *config.Config
	sup     *supervisor.Supervisor
	db      *sqlx.DB
	builder *Builder

	startMu sync.Mutex
	mu      sync.Mutex
	live    map[sessionKey]*Session
}

func NewManager(cfg *config.Config, sup *supervisor.Supervisor, db *sqlx.DB) *Manager {
	return &Manager{
		cfg:     cfg,
		sup:     sup,
		db:      db,
		builder: NewBuilder(sup, cfg),
		live:    map[sessionKey]*Session{},
	}
}

// Start returns the user's running session for an assessment, creating the
// environment and container on first use. The first call can take minutes
// while the assessment image and harness are built; later calls are cheap.
func (m *Manager) Start(ctx context.Context, user models.User, assessmentID int64) (*Session, error) {
	m.startMu.Lock()
	defer m.startMu.Unlock()

	if s, ok := m.Current(user.ID, assessmentID); ok {
		return s, nil
	}

	detail, err := database.GetAssessmentDetail(ctx, m.db, assessmentID)
	if err != nil {
		if errors.Is(err, database.ErrNotFound) {
			return nil, ErrAssessmentGone
		}
		return nil, err
	}
	if detail.Kind != kindLeetcode {
		return nil, ErrUnsupported
	}
	chapter, ok := firstRunnableChapter(detail.Chapters)
	if !ok {
		return nil, ErrNotRunnable
	}
	sha, err := database.GetAssessmentMainCommit(ctx, m.db, assessmentID)
	if err != nil {
		return nil, err
	}
	if sha == "" {
		return nil, ErrNotSynced
	}

	env, err := m.builder.Build(ctx, assessmentID, m.assessmentRepoDir(assessmentID), sha)
	if err != nil {
		return nil, err
	}
	chapterDir := chapterDirPath(chapter)
	build, ok := env.chapter(chapterDir)
	if !ok {
		return nil, fmt.Errorf("%w: chapter %s is missing from the built environment", ErrNotRunnable, chapterDir)
	}
	taskFile := path.Join(chapterDir, build.Task)

	sessionID := randomHex(16)
	spec := supervisor.ContainerSpec{
		Runtime:         "runsc",
		Cmd:             []string{"sleep", "infinity"},
		WorkDir:         "/",
		NetworkDisabled: true,
		MemoryBytes:     m.cfg.AssessmentEnv.MemoryBytes,
		NanoCPUs:        m.cfg.AssessmentEnv.NanoCPUs,
		PidsLimit:       m.cfg.AssessmentEnv.PidsLimit,
		Labels: map[string]string{
			sessionLabelRole:        sessionRole,
			"codingtest.assessment": strconv.FormatInt(assessmentID, 10),
			"codingtest.user":       strconv.FormatInt(user.ID, 10),
			"codingtest.session":    sessionID,
		},
	}
	if err := m.sup.CreateContainer(env.Image, sessionID, spec); err != nil {
		return nil, err
	}
	if err := m.sup.StartContainer(sessionID); err != nil {
		m.sup.RemoveContainer(sessionID)
		return nil, err
	}

	session := &Session{
		ID:              sessionID,
		UserID:          user.ID,
		AssessmentID:    assessmentID,
		AssessmentTitle: detail.Title,
		Kind:            detail.Kind,
		ChapterID:       chapter.ID,
		ChapterTitle:    chapter.Title,
		ChapterDir:      chapterDir,
		TaskFile:        taskFile,
		TimeLimit:       detail.TimeLimitMinutes,
		Deadline:        time.Now().Add(m.sessionTTL(detail.TimeLimitMinutes)),
		CreatedAt:       time.Now(),
		specs:           build.Specs,
	}
	content, err := m.sup.ReadFile(sessionID, "/workspace/"+taskFile)
	if err != nil {
		m.discard(session)
		return nil, fmt.Errorf("read the task file from the environment: %w", err)
	}
	session.Content = string(content)

	m.mu.Lock()
	m.live[sessionKey{user.ID, assessmentID}] = session
	m.mu.Unlock()
	log.Printf("sessions: started %s for user %d on assessment %d (%s, deadline %s)",
		sessionID, user.ID, assessmentID, env.Image, session.Deadline.Format(time.RFC3339))
	return session, nil
}

// Current returns the user's live session for an assessment. Expired
// sessions are ended here so callers never see a dead one.
func (m *Manager) Current(userID, assessmentID int64) (*Session, bool) {
	key := sessionKey{userID, assessmentID}
	m.mu.Lock()
	session, ok := m.live[key]
	if ok && time.Now().After(session.Deadline) {
		delete(m.live, key)
		m.mu.Unlock()
		go m.discard(session)
		return nil, false
	}
	m.mu.Unlock()
	return session, ok
}

// WriteFile writes the editor contents into the session's task file.
func (m *Manager) WriteFile(session *Session, content string) error {
	session.mu.Lock()
	defer session.mu.Unlock()
	if err := m.checkAlive(session); err != nil {
		return err
	}
	if len(content) > maxFileBytes {
		return ErrFileTooLarge
	}
	if err := m.sup.WriteFile(session.ID, "/workspace/"+session.TaskFile, []byte(content)); err != nil {
		return err
	}
	session.Content = content
	return nil
}

// RunTests executes the chapter's compiled checks against the task file.
func (m *Manager) RunTests(ctx context.Context, session *Session) (TestReport, error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	report := TestReport{Results: []TestResult{}, AllPassed: true}
	if err := m.checkAlive(session); err != nil {
		return report, err
	}
	if len(session.specs) == 0 {
		return report, ErrNotRunnable
	}

	var raw strings.Builder
	for _, spec := range session.specs {
		result, err := m.sup.Exec(session.ID, supervisor.ExecSpec{
			Cmd:     []string{"/workspace/" + spec.Exe},
			WorkDir: "/workspace/" + spec.WorkDir,
			Timeout: m.cfg.AssessmentEnv.ExecTimeout,
		})
		if result.Output != "" {
			raw.WriteString(result.Output)
			if !strings.HasSuffix(result.Output, "\n") {
				raw.WriteByte('\n')
			}
		}
		if err != nil {
			report.Raw = raw.String()
			return report, err
		}
		if result.ExitCode != 0 {
			report.AllPassed = false
		}
		report.Results = append(report.Results, parseChecks(result.Output)...)
	}
	for _, result := range report.Results {
		if !result.Passed {
			report.AllPassed = false
		}
	}
	report.Raw = raw.String()
	return report, nil
}

// Submit persists the task file as the user's solution for the assessment
// and ends the session.
func (m *Manager) Submit(ctx context.Context, user models.User, session *Session, public bool) (models.Solution, error) {
	session.mu.Lock()
	defer session.mu.Unlock()
	if err := m.checkAlive(session); err != nil {
		return models.Solution{}, err
	}

	content, err := m.sup.ReadFile(session.ID, "/workspace/"+session.TaskFile)
	if err != nil {
		return models.Solution{}, err
	}
	sha, err := repos.CommitFiles(m.solutionRepoDir(session.AssessmentID, user.ID),
		map[string][]byte{session.TaskFile: content},
		solutionMessage(user, session),
		object.Signature{Name: user.Username, Email: user.Email, When: time.Now()})
	if err != nil {
		return models.Solution{}, err
	}
	solution, err := database.SaveSolution(ctx, m.db, session.AssessmentID, user.ID, sha, public)
	if err != nil {
		return models.Solution{}, err
	}

	m.mu.Lock()
	delete(m.live, sessionKey{user.ID, session.AssessmentID})
	m.mu.Unlock()
	m.discard(session)
	log.Printf("sessions: user %d submitted assessment %d as %s", user.ID, session.AssessmentID, sha)
	return solution, nil
}

// Abandon ends a session without persisting anything.
func (m *Manager) Abandon(session *Session) error {
	m.mu.Lock()
	delete(m.live, sessionKey{session.UserID, session.AssessmentID})
	m.mu.Unlock()
	return m.discard(session)
}

// Janitor ends sessions whose deadline passed. Run it in a goroutine for the
// lifetime of the process.
func (m *Manager) Janitor(ctx context.Context) {
	ticker := time.NewTicker(janitorInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()
			for _, session := range m.snapshot() {
				if now.After(session.Deadline) {
					log.Printf("sessions: session %s passed its deadline", session.ID)
					if err := m.Abandon(session); err != nil {
						log.Printf("sessions: end expired session %s: %v", session.ID, err)
					}
				}
			}
		}
	}
}

// CleanupOrphans removes session containers left behind by a previous
// backend process. Run it once at boot.
func (m *Manager) CleanupOrphans(ctx context.Context) {
	containers, err := m.sup.ListContainers(ctx, map[string]string{sessionLabelRole: sessionRole})
	if err != nil {
		log.Printf("sessions: list leftover environments: %v", err)
		return
	}
	for _, container := range containers {
		if err := m.sup.RemoveDockerContainer(ctx, container.ID); err != nil {
			log.Printf("sessions: remove leftover environment %s: %v", container.ID, err)
			continue
		}
		log.Printf("sessions: removed leftover environment %s", container.ID)
	}
}

// checkAlive rejects work on a session that passed its deadline.
func (m *Manager) checkAlive(session *Session) error {
	if time.Now().After(session.Deadline) {
		return ErrSessionGone
	}
	return nil
}

// discard kills the container and forgets the session; callers have already
// removed it from the map.
func (m *Manager) discard(session *Session) error {
	if err := m.sup.RemoveContainer(session.ID); err != nil {
		log.Printf("sessions: remove container of session %s: %v", session.ID, err)
		return err
	}
	return nil
}

// snapshot lists the live sessions.
func (m *Manager) snapshot() []*Session {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*Session, 0, len(m.live))
	for _, session := range m.live {
		out = append(out, session)
	}
	return out
}

// sessionTTL is the assessment's own limit, falling back to the configured
// default when the assessment has none.
func (m *Manager) sessionTTL(timeLimitMinutes int) time.Duration {
	if timeLimitMinutes > 0 {
		return time.Duration(timeLimitMinutes) * time.Minute
	}
	return m.cfg.AssessmentEnv.SessionTTL
}

func (m *Manager) assessmentRepoDir(assessmentID int64) string {
	return filepath.Join(m.cfg.GitReposDir, filepath.FromSlash(models.AssessmentRepoID(assessmentID)+".git"))
}

func (m *Manager) solutionRepoDir(assessmentID, userID int64) string {
	return filepath.Join(m.cfg.GitReposDir, filepath.FromSlash(models.SolutionRepoID(assessmentID, userID)+".git"))
}

// firstRunnableChapter picks the earliest chapter that declares a task file.
func firstRunnableChapter(chapters []models.Chapter) (models.Chapter, bool) {
	var picked models.Chapter
	found := false
	for _, chapter := range chapters {
		if strings.TrimSpace(chapter.TaskFile) == "" {
			continue
		}
		if !found || chapter.Position < picked.Position {
			picked = chapter
			found = true
		}
	}
	return picked, found
}

// chapterDirPath mirrors the chapter directory naming of the layout:
// chapters/<position>_<title>.
func chapterDirPath(chapter models.Chapter) string {
	return fmt.Sprintf("chapters/%d_%s", chapter.Position, chapter.Title)
}

func solutionMessage(user models.User, session *Session) string {
	return fmt.Sprintf("solution: %s by %s\n\nAssessment: %s (id %d)\nChapter: %s\nTask file: %s\nSubmitted at: %s\n",
		session.AssessmentTitle, user.Username, session.AssessmentTitle, session.AssessmentID,
		session.ChapterTitle, session.TaskFile, time.Now().UTC().Format(time.RFC3339))
}

func randomHex(bytes int) string {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		panic(err)
	}
	return hex.EncodeToString(buf)
}
