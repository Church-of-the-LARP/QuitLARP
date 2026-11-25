package models

import "time"

// TestCheck is one named check from a run-tests invocation.
type TestCheck struct {
	Name   string `json:"name"`
	Passed bool   `json:"passed"`
	Detail string `json:"detail"`
}

// AssessmentTestRun is the last stored checks run for one user and assessment.
type AssessmentTestRun struct {
	ID           int64       `json:"id"`
	AssessmentID int64       `json:"assessmentId"`
	Results      []TestCheck `json:"results"`
	AllPassed    bool        `json:"allPassed"`
	CreatedAt    time.Time   `json:"createdAt"`
}

// SolutionFile is one text file of a submitted solution.
type SolutionFile struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// SolutionRef summarizes a stored coding solution.
type SolutionRef struct {
	CommitSHA string         `json:"commitSha"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	Files     []SolutionFile `json:"files"`
}

// EventResultParticipant is one assigned participant's status on a coding
// event task: their latest submission and latest check run.
type EventResultParticipant struct {
	UserID   int64              `json:"userId"`
	Username string             `json:"username"`
	Solution *SolutionRef       `json:"solution"`
	TestRun  *AssessmentTestRun `json:"testRun"`
}

// EventTaskResults is the host-facing result view of one event task. Unbound
// tasks report attempts (workspace, commit, deployment); coding tasks report
// each participant's latest solution and checks.
type EventTaskResults struct {
	ID           int64                    `json:"id"`
	AssessmentID int64                    `json:"assessmentId"`
	Kind         string                   `json:"kind"`
	Title        string                   `json:"title"`
	Attempts     []EventAttempt           `json:"attempts"`
	Participants []EventResultParticipant `json:"participants"`
}

// EventResults is the response of GET /api/v1/events/{id}/results.
type EventResults struct {
	Tasks []EventTaskResults `json:"tasks"`
}
