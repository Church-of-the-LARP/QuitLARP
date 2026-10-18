package models

import "time"

// ScheduledAssessment is one entry of the caller's upcoming-tests list: an
// assessment they planned to take, and when.
type ScheduledAssessment struct {
	ID               int64      `json:"id" db:"id" doc:"Unique database identifier of the schedule entry"`
	AssessmentID     int64      `json:"assessmentId" db:"assessment_id" doc:"The scheduled assessment"`
	Title            string     `json:"title" db:"title" doc:"Assessment title"`
	Difficulty       Difficulty `json:"difficulty" db:"difficulty" enum:"easy,medium,hard" doc:"Difficulty rating"`
	TimeLimitMinutes int        `json:"timeLimitMinutes" db:"time_limit_minutes" example:"120" doc:"Time limit for the whole assessment, in minutes"`
	TestCount        int        `json:"testCount" db:"test_count" doc:"Number of hidden tests (the dashboard's question count)"`
	ScheduledAt      time.Time  `json:"scheduledAt" db:"scheduled_at" doc:"When the user plans to take the assessment"`
}
