// Package sessions runs the sandboxed environments candidates solve
// assessments in: it builds each assessment's image from its git repository,
// compiles the chapter test harness, spawns one container per solving user
// and runs the chapter checks on demand.
package sessions

import (
	"errors"
	"sync"
	"time"
)

// kindLeetcode is the only assessment kind implemented today: a single
// candidate-edited file per chapter, exercised by a UFT harness.
const kindLeetcode = "leetcode"

// maxFileBytes caps how much of the task file a client may write. The editor
// never needs more; the cap keeps a session from being used as storage.
const maxFileBytes = 512 << 10

// Errors the handlers map to HTTP statuses.
var (
	ErrUnsupported    = errors.New("this assessment kind is not supported yet")
	ErrNotRunnable    = errors.New("this assessment has no solvable chapter")
	ErrNotSynced      = errors.New("the assessment repository has not been pushed to yet")
	ErrSessionGone    = errors.New("the solving session is no longer running")
	ErrTaskFileOnly   = errors.New("only the declared task file can be written")
	ErrFileTooLarge   = errors.New("the file is larger than the editor allows")
	ErrAssessmentGone = errors.New("the assessment no longer exists")
)

// SpecRun is one check bundle of a chapter: the compiled harness executable
// and the directory it must run in.
type SpecRun struct {
	File    string `json:"file"`    // spec source, slash path from the repository root
	Exe     string `json:"exe"`     // compiled executable, slash path from the repository root
	WorkDir string `json:"workDir"` // directory the executable runs in, slash path from the repository root
}

// Session is one live solving environment.
type Session struct {
	ID              string
	UserID          int64
	AssessmentID    int64
	AssessmentTitle string
	Kind            string
	ChapterID       int64
	ChapterTitle    string
	ChapterDir      string // slash path of the chapter from the repository root
	TaskFile        string // slash path of the candidate file from the repository root
	TimeLimit       int    // assessment limit in minutes, zero when unlimited
	Deadline        time.Time
	CreatedAt       time.Time

	Content string

	specs []SpecRun
	mu    sync.Mutex
}

// TestResult is one check reported by the chapter harness.
type TestResult struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// TestReport is the outcome of one "run tests" request.
type TestReport struct {
	Results   []TestResult `json:"results"`
	Raw       string       `json:"raw"`
	AllPassed bool         `json:"allPassed"`
}

// Environment is a built assessment: the image to spawn sessions from plus
// the per-chapter harness layout discovered while building it.
type Environment struct {
	AssessmentID int64          `json:"assessmentId"`
	SHA          string         `json:"sha"`
	Image        string         `json:"image"`
	Dir          string         `json:"dir"`
	Chapters     []ChapterBuild `json:"chapters"`
}

// ChapterBuild is the built harness of one chapter.
type ChapterBuild struct {
	Dir   string    `json:"dir"`   // chapters/1_two-sum
	Title string    `json:"title"` // two-sum
	Task  string    `json:"task"`  // chapter-relative path of the candidate file
	Specs []SpecRun `json:"specs"`
}

// chapter returns the build entry for a chapter directory, if any.
func (e *Environment) chapter(dir string) (ChapterBuild, bool) {
	for _, ch := range e.Chapters {
		if ch.Dir == dir {
			return ch, true
		}
	}
	return ChapterBuild{}, false
}
