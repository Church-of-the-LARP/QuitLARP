package models

import "time"

// Difficulty is the rating of an assessment, mirroring the three LeetCode
// tiers. Kept as a string for readable payloads; validated on the way in.
type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

// Valid reports whether d is one of the known difficulties.
func (d Difficulty) Valid() bool {
	switch d {
	case DifficultyEasy, DifficultyMedium, DifficultyHard:
		return true
	}
	return false
}

// ParseDifficulty converts a string to a Difficulty.
func ParseDifficulty(s string) (Difficulty, bool) {
	d := Difficulty(s)
	return d, d.Valid()
}

// CodeFile is a source file attached to an assessment: the base template the
// candidate starts from, or a hidden testing file written by the author.
// Content is stored in the database for now; swapping in a blob store later
// keeps this payload shape intact.
type CodeFile struct {
	FileName string `json:"fileName" db:"file_name" doc:"File name, including extension"`
	Content  string `json:"content" db:"file_content" doc:"File contents"`
}

// Tag is a label shared across assessments.
type Tag struct {
	ID   int64  `json:"id" db:"id" doc:"Unique database identifier"`
	Name string `json:"name" db:"name" doc:"Tag label, unique case-insensitively"`
}

// TagCount is a tag plus how many assessments carry it (GET /tags).
type TagCount struct {
	ID              int64  `json:"id" db:"id" doc:"Unique database identifier"`
	Name            string `json:"name" db:"name" doc:"Tag label, unique case-insensitively"`
	AssessmentCount int64  `json:"assessmentCount" db:"assessment_count" doc:"Number of assessments carrying this tag"`
}

// AuthorRef is the public, minimal representation of an assessment's author.
type AuthorRef struct {
	ID       int64  `json:"id" doc:"Unique database identifier"`
	Username string `json:"username" doc:"Unique display/account name"`
}

// AssessmentSummary is one row of GET /assessments; full content (template,
// chapters, tests) lives on GET /assessments/{id}.
type AssessmentSummary struct {
	ID               int64      `json:"id" db:"id" doc:"Unique database identifier"`
	Title            string     `json:"title" db:"title" doc:"Assessment title"`
	Description      string     `json:"description" db:"description" doc:"What the candidate has to build"`
	Difficulty       Difficulty `json:"difficulty" db:"difficulty" enum:"easy,medium,hard" doc:"Difficulty rating"`
	TimeLimitMinutes int        `json:"timeLimitMinutes" db:"time_limit_minutes" example:"120" doc:"Time limit for the whole assessment, in minutes"`
	TemplateFileName string     `json:"templateFileName" db:"template_file_name" doc:"Name of the base template file the candidate starts from"`
	Author           *AuthorRef `json:"author" doc:"Author of the assessment; null when the account was deleted"`
	Tags             []Tag      `json:"tags" doc:"Tags attached to the assessment"`
	CreatedAt        time.Time  `json:"createdAt" db:"created_at" doc:"Creation time"`
	UpdatedAt        time.Time  `json:"updatedAt" db:"updated_at" doc:"Last modification time"`
}

// Chapter organizes the content of an assessment. Positions are 1-based
// ordering hints; ordering is (position, id).
type Chapter struct {
	ID               int64     `json:"id" db:"id" doc:"Unique database identifier"`
	AssessmentID     int64     `json:"assessmentId" db:"assessment_id" doc:"The assessment this chapter belongs to"`
	Position         int       `json:"position" db:"position" example:"1" doc:"1-based ordering hint within the assessment"`
	Title            string    `json:"title" db:"title" doc:"Chapter title"`
	Description      string    `json:"description" db:"description" doc:"Chapter body text"`
	TimeLimitMinutes int       `json:"timeLimitMinutes" db:"time_limit_minutes" example:"30" doc:"Time limit for this chapter, in minutes"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at" doc:"Creation time"`
	UpdatedAt        time.Time `json:"updatedAt" db:"updated_at" doc:"Last modification time"`
}

// TestSummary is everything a candidate may see about a hidden test: the
// name and description. The testing file itself is never exposed here.
type TestSummary struct {
	ID          int64  `json:"id" db:"id" doc:"Unique database identifier"`
	Name        string `json:"name" db:"name" doc:"Test name"`
	Description string `json:"description" db:"description" doc:"What this test verifies"`
}

// Test is the full representation of a hidden test, including the testing
// file. Only the author (or an admin) may read it.
type Test struct {
	TestSummary
	AssessmentID int64     `json:"assessmentId" db:"assessment_id" doc:"The assessment this test belongs to"`
	File         CodeFile  `json:"file" doc:"The hidden testing file written by the author"`
	CreatedAt    time.Time `json:"createdAt" db:"created_at" doc:"Creation time"`
	UpdatedAt    time.Time `json:"updatedAt" db:"updated_at" doc:"Last modification time"`
}

// Assessment is the full public representation returned by GET
// /assessments/{id}: everything the candidate needs to start working, plus
// the hidden tests as name/description summaries only.
type Assessment struct {
	ID               int64         `json:"id" db:"id" doc:"Unique database identifier"`
	Title            string        `json:"title" db:"title" doc:"Assessment title"`
	Description      string        `json:"description" db:"description" doc:"What the candidate has to build"`
	Difficulty       Difficulty    `json:"difficulty" db:"difficulty" enum:"easy,medium,hard" doc:"Difficulty rating"`
	TimeLimitMinutes int           `json:"timeLimitMinutes" db:"time_limit_minutes" example:"120" doc:"Time limit for the whole assessment, in minutes"`
	Template         CodeFile      `json:"template" doc:"Base template file the candidate starts from"`
	Author           *AuthorRef    `json:"author" doc:"Author of the assessment; null when the account was deleted"`
	Tags             []Tag         `json:"tags" doc:"Tags attached to the assessment"`
	Chapters         []Chapter     `json:"chapters" doc:"Chapters of the assessment, ordered by position"`
	Tests            []TestSummary `json:"tests" doc:"Hidden tests, name and description only"`
	CreatedAt        time.Time     `json:"createdAt" db:"created_at" doc:"Creation time"`
	UpdatedAt        time.Time     `json:"updatedAt" db:"updated_at" doc:"Last modification time"`
}

// ChapterDraft is a chapter as supplied when creating an assessment or
// appending a chapter to an existing one.
type ChapterDraft struct {
	Title            string `json:"title" doc:"Chapter title"`
	Description      string `json:"description" doc:"Chapter body text"`
	TimeLimitMinutes int    `json:"timeLimitMinutes" example:"30" doc:"Time limit for this chapter, in minutes"`
}

// TestDraft is a hidden test as supplied when creating an assessment or
// adding a test to an existing one.
type TestDraft struct {
	Name        string   `json:"name" doc:"Test name"`
	Description string   `json:"description" doc:"What this test verifies"`
	File        CodeFile `json:"file" doc:"The hidden testing file written by the author"`
}

// AssessmentDraft is the full payload of POST /assessments. Tag names that
// do not exist yet are created on the fly; duplicates are ignored.
type AssessmentDraft struct {
	Title            string         `json:"title" doc:"Assessment title"`
	Description      string         `json:"description" doc:"What the candidate has to build"`
	Difficulty       Difficulty     `json:"difficulty" enum:"easy,medium,hard" doc:"Difficulty rating"`
	TimeLimitMinutes int            `json:"timeLimitMinutes" example:"120" doc:"Time limit for the whole assessment, in minutes"`
	Template         CodeFile       `json:"template" doc:"Base template file the candidate starts from"`
	Tags             []string       `json:"tags,omitempty" doc:"Tag labels; unknown ones are created automatically"`
	Chapters         []ChapterDraft `json:"chapters,omitempty" doc:"Chapters, in the order they should appear"`
	Tests            []TestDraft    `json:"tests,omitempty" doc:"Hidden tests with their testing files"`
}
