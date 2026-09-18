package models

import "time"

// AttemptStatus is the three-state lifecycle of an assessment attempt.
// "Waiting for the schedule" is in-memory Runner state, not a 4th status.
type AttemptStatus string

const (
	AttemptPending AttemptStatus = "pending"
	AttemptRunning AttemptStatus = "running"
	AttemptEnded   AttemptStatus = "ended"
)

// Valid reports whether s is one of the known attempt statuses.
func (s AttemptStatus) Valid() bool {
	switch s {
	case AttemptPending, AttemptRunning, AttemptEnded:
		return true
	}
	return false
}

// Invite is a host's scheduled sitting for a group. scheduledStartAt is
// always set — there is no unscheduled flavor. timeLimitMinutes/maxUses are
// nil when the sitting uses the assessment's own limit / has no headcount
// cap.
type Invite struct {
	ID               int64     `json:"id" db:"id" doc:"Unique database identifier"`
	AssessmentID     int64     `json:"assessmentId" db:"assessment_id" doc:"The assessment this invite schedules a sitting for"`
	HostID           *int64    `json:"hostId" db:"host_id" doc:"The user who created this invite; null when that account was deleted"`
	InviteCode       string    `json:"inviteCode" db:"invite_code" doc:"Shareable code acceptors use to register"`
	ScheduledStartAt time.Time `json:"scheduledStartAt" db:"scheduled_start_at" doc:"The instant every acceptor's clock is anchored to"`
	TimeLimitMinutes *int      `json:"timeLimitMinutes" db:"time_limit_minutes" example:"90" doc:"Overrides the assessment's own time limit for this sitting; null to use the assessment's"`
	MaxUses          *int      `json:"maxUses" db:"max_uses" example:"20" doc:"Cap on distinct acceptors; null for unlimited"`
	UsesCount        int       `json:"usesCount" db:"uses_count" doc:"Number of distinct acceptors so far"`
	CreatedAt        time.Time `json:"createdAt" db:"created_at" doc:"Creation time"`
}

// Attempt is one user's one run through one assessment. inviteId is nil for
// a solo, self-started attempt.
type Attempt struct {
	ID               int64         `json:"id" db:"id" doc:"Unique database identifier"`
	AssessmentID     int64         `json:"assessmentId" db:"assessment_id" doc:"The assessment being attempted"`
	InviteID         *int64        `json:"inviteId" db:"invite_id" doc:"The invite this attempt was created from; null for a solo attempt"`
	UserID           int64         `json:"userId" db:"user_id" doc:"The one owner of this attempt"`
	Status           AttemptStatus `json:"status" db:"status" enum:"pending,running,ended" doc:"Lifecycle status"`
	TimeLimitMinutes int           `json:"timeLimitMinutes" db:"time_limit_minutes" example:"90" doc:"Time limit copied at creation time"`
	CurrentChapterID *int64        `json:"currentChapterId" db:"current_chapter_id" doc:"The chapter the candidate is currently on; null before starting"`
	StartedAt        *time.Time    `json:"startedAt" db:"started_at" doc:"When the clock actually started; null while pending"`
	EndedAt          *time.Time    `json:"endedAt" db:"ended_at" doc:"When the attempt ended; null while active"`
}
